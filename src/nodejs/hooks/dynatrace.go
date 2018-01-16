package hooks

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

type Command interface {
	Execute(string, io.Writer, io.Writer, string, ...string) error
}

type DynatraceHook struct {
	libbuildpack.DefaultHook
	Log     *libbuildpack.Logger
	Command Command
}

type Service struct {
	Name        string                 `json:"name"`
	Credentials map[string]interface{} `json:"credentials"`
}

type DynatraceService struct {
	Name          string
	ApiUrl        string
	ApiToken      string
	EnvironmentId string
	SkipErrors    bool
}

func init() {
	logger := libbuildpack.NewLogger(os.Stdout)
	command := &libbuildpack.Command{}

	libbuildpack.AddHook(DynatraceHook{
		Log:     logger,
		Command: command,
	})
}

func (h DynatraceHook) AfterCompile(stager *libbuildpack.Stager) error {
	h.Log.Debug("Checking for enabled dynatrace service...")

	credentials, err := h.dtCredentials()
	if err != nil {
		h.Log.Debug(err.Error())
		return nil
	}

	h.Log.Info("Dynatrace service credentials found. Setting up Dynatrace PaaS agent.")

	url := credentials.ApiUrl + "/v1/deployment/installer/agent/unix/paas-sh/latest?include=nodejs&include=process&bitness=64&Api-Token=" + credentials.ApiToken
	installerPath := filepath.Join(os.TempDir(), "paasInstaller.sh")

	h.Log.Debug("Downloading '%s' to '%s'", url, installerPath)
	err = h.downloadFile(url, installerPath)
	if err != nil {
		if credentials.SkipErrors {
			h.Log.Warning("Error during installer download, skipping installation")
			return nil
		}
		return err
	}

	h.Log.Debug("Making %s executable...", installerPath)
	os.Chmod(installerPath, 0755)

	h.Log.BeginStep("Starting Dynatrace PaaS agent installer")

	if os.Getenv("BP_DEBUG") != "" {
		err = h.Command.Execute("", os.Stdout, os.Stderr, installerPath, stager.BuildDir())
	} else {
		err = h.Command.Execute("", ioutil.Discard, ioutil.Discard, installerPath, stager.BuildDir())
	}
	if err != nil {
		return err
	}

	h.Log.Info("Dynatrace PaaS agent installed.")

	dynatraceEnvName := "dynatrace-env.sh"
	installDir := "dynatrace/oneagent"
	dynatraceEnvPath := filepath.Join(stager.DepDir(), "profile.d", dynatraceEnvName)
	agentLibPath, err := h.agentPath(filepath.Join(stager.BuildDir(), installDir))
	if err != nil {
		h.Log.Error("Manifest handling failed!")
		return err
	}

	agentLibPath = filepath.Join(installDir, agentLibPath)

	_, err = os.Stat(filepath.Join(stager.BuildDir(), agentLibPath))
	if os.IsNotExist(err) {
		h.Log.Error("Agent library (%s) not found!", filepath.Join(installDir, agentLibPath))
		return err
	}

	h.Log.BeginStep("Setting up Dynatrace PaaS agent injection...")
	h.Log.Debug("Copy %s to %s", dynatraceEnvName, dynatraceEnvPath)
	err = libbuildpack.CopyFile(filepath.Join(stager.BuildDir(), installDir, dynatraceEnvName), dynatraceEnvPath)
	if err != nil {
		return err
	}

	h.Log.Debug("Open %s for modification...", dynatraceEnvPath)
	f, err := os.OpenFile(dynatraceEnvPath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return err
	}

	defer f.Close()

	h.Log.Debug("Write LD_PRELOAD...")
	_, err = f.WriteString("\nexport LD_PRELOAD=${HOME}/" + agentLibPath)
	if err != nil {
		return err
	}

	h.Log.Debug("Write DT_HOST_ID...")
	_, err = f.WriteString("\nexport DT_HOST_ID=" + h.appName() + "_${CF_INSTANCE_INDEX}")
	if err != nil {
		return err
	}

	h.Log.Info("Dynatrace PaaS agent injection is set up.")

	return nil
}

func (h DynatraceHook) dtCredentials() (DynatraceService, error) {
	var dynatraceService DynatraceService
	var vcapServices map[string][]Service

	err := json.Unmarshal([]byte(os.Getenv("VCAP_SERVICES")), &vcapServices)
	if err != nil {
		// we don't know whether skiperror has been set so we log and continue
		h.Log.Warning("Cannot unmarshal VCAP_SERVICES")
		return dynatraceService, errors.New("Cannot unmarshal VCAP_SERVICES")
	}

	var detectedServices []DynatraceService

	for _, services := range vcapServices {
		for _, service := range services {
			dynatraceService, err = h.parseDynatraceService(service)
			if err == nil {
				detectedServices = append(detectedServices, dynatraceService)
			}
		}
	}

	if len(detectedServices) == 0 {
		return dynatraceService, errors.New("Dynatrace service credentials not found")
	} else if len(detectedServices) > 1 {
		h.Log.Warning("More than one matching service found!")
		return dynatraceService, errors.New("More than one matching service found")
	}
	return detectedServices[0], nil
}

func (h DynatraceHook) parseDynatraceService(service Service) (DynatraceService, error) {
	var dynatraceService DynatraceService
	if !strings.Contains(service.Name, "dynatrace") {
		return dynatraceService, errors.New("")
	}

	dynatraceService.Name = service.Name

	if environmentid, ok := service.Credentials["environmentid"]; ok {
		dynatraceService.EnvironmentId = environmentid.(string)
	} else {
		h.Log.Warning("Service '%s' is missing mandatory property 'environmentid'", dynatraceService.Name)
		return dynatraceService, errors.New("")
	}

	if apitoken, ok := service.Credentials["apitoken"]; ok {
		dynatraceService.ApiToken = apitoken.(string)
	} else {
		h.Log.Warning("Service '%s' is missing mandatory property 'apitoken'", dynatraceService.Name)
		return dynatraceService, errors.New("")
	}

	if apiurl, ok := service.Credentials["apiurl"]; ok {
		dynatraceService.ApiUrl = apiurl.(string)
	} else {
		dynatraceService.ApiUrl = "https://" + dynatraceService.EnvironmentId + ".live.dynatrace.com/api"
	}

	if skipErrors, ok := service.Credentials["skiperrors"]; ok {
		skipErrorsBool, err := strconv.ParseBool(skipErrors.(string))

		if err == nil {
			dynatraceService.SkipErrors = skipErrorsBool
		} else {
			h.Log.Warning("Invalid value for property 'skiperrors'")
		}
	}

	return dynatraceService, nil
}

func (h DynatraceHook) appName() string {
	var application struct {
		Name string `json:"name"`
	}
	err := json.Unmarshal([]byte(os.Getenv("VCAP_APPLICATION")), &application)
	if err != nil {
		return ""
	}

	return application.Name
}

func (h DynatraceHook) downloadFile(url, path string) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}

	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return errors.New("Download returned with status " + resp.Status)
	}

	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func (h DynatraceHook) agentPath(installDir string) (string, error) {
	manifestPath := filepath.Join(installDir, "manifest.json")

	type Binary struct {
		Path       string `json:"path"`
		Md5        string `json:"md5"`
		Version    string `json:"version"`
		Binarytype string `json:"binarytype,omitempty"`
	}

	type Architecture map[string][]Binary
	type Technologies map[string]Architecture

	type Manifest struct {
		Tech Technologies `json:"technologies"`
		Ver  string       `json:"version"`
	}

	var manifest Manifest

	raw, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		return "", err
	}

	err = json.Unmarshal(raw, &manifest)
	if err != nil {
		return "", err
	}

	for _, binary := range manifest.Tech["process"]["linux-x86-64"] {
		if binary.Binarytype == "primary" {
			return binary.Path, nil
		}
	}

	return "", errors.New("No primary binary for process agent found")
}

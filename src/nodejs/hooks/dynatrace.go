package hooks

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
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

	credentials := h.dtCredentials()
	if credentials == nil {
		h.Log.Debug("Dynatrace service not found!")
		return nil
	}

	h.Log.Info("Dynatrace service found. Setting up Dynatrace PaaS agent.")

	apiurl, present := credentials["apiurl"]
	if !present && credentials["environmentid"] != "" {
		apiurl = "https://" + credentials["environmentid"] + ".live.dynatrace.com/api"
	}

	if apiurl == "" {
		return errors.New("'environmentid' or 'apiurl' has to be specified in the service credentials!")
	}

	if credentials["apitoken"] == "" {
		return errors.New("'apitoken' has to be specified in the service credentials!")
	}

	url := apiurl + "/v1/deployment/installer/agent/unix/paas-sh/latest?include=nodejs&bitness=64&Api-Token=" + credentials["apitoken"]
	installerPath := filepath.Join(os.TempDir(), "paasInstaller.sh")

	h.Log.Debug("Downloading '%s' to '%s'", url, installerPath)
	err := h.downloadFile(url, installerPath)
	if err != nil {
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
		h.Log.Error("manifest.json not found in %s!", installDir)
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

func (h DynatraceHook) dtCredentials() map[string]string {
	type Service struct {
		Name        string            `json:"name"`
		Credentials map[string]string `json:"credentials"`
	}
	var vcapServices map[string][]Service

	err := json.Unmarshal([]byte(os.Getenv("VCAP_SERVICES")), &vcapServices)
	if err != nil {
		return nil
	}

	var detectedServices []Service

	for _, services := range vcapServices {
		for _, service := range services {
			if strings.Contains(service.Name, "dynatrace") &&
					service.Credentials["environmentid"] != "" &&
					service.Credentials["apitoken"] != "" {
				detectedServices = append(detectedServices, service)
			}
		}
	}

	if len(detectedServices) == 1 {
		h.Log.Debug("Found one matching service: %s", detectedServices[0].Name)
		return detectedServices[0].Credentials
	} else if len(detectedServices) > 1 {
		h.Log.Warning("More than one matching service found!")
	}

	return nil
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
	manifestPath := filepath.Join(installDir, "/manifest.json")

	type Binary struct {
		Path string `json:"path"`
		Md5 string `json:"md5"`
		Version string `json:"version"`
		Binarytype string `json:"binarytype,omitemtpy"`
	}

	type Architecture map[string][]Binary
	type Technologies map[string]Architecture

	type Manifest struct {
		Tech Technologies`json:"technologies"`
		Ver string `json:"version"`
	}

	var m Manifest

	raw, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		return "", err
	}

	err = json.Unmarshal(raw, &m) 
	if err != nil {
		return "", err
	}

	for _, binary := range m.Tech["process"]["linux-x86-64"] {
		if binary.Binarytype ==	"primary" {
			return binary.Path, nil
		}
	}

	return "", errors.New("No primary binary for process agent found!")
}

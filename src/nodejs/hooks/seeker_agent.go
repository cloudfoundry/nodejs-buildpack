package hooks

import (
	"archive/zip"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cloudfoundry/libbuildpack"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	EntryPointFile    = "SEEKER_APP_ENTRY_POINT"
	agentDownloadPath = "/rest/api/latest/installers/agents/binaries/NODEJS"
	SeekerRequire     = "require('./seeker/node_modules/@synopsys-sig/seeker');\n"
)

var pattern = regexp.MustCompile(`require\(['"].*@synopsys-sig/seeker['"]\)`)

type SeekerCommand interface {
	Execute(dir string, stdout io.Writer, stderr io.Writer, program string, args ...string) error
}

type Downloader interface {
	DownloadFile(url, destFile string) error
}
type Unzipper interface {
	Unzip(zipFile, absoluteFolderPath string) error
}

type SeekerAfterCompileHook struct {
	libbuildpack.DefaultHook
	Log                *libbuildpack.Logger
	serviceCredentials *SeekerCredentials
	Command            SeekerCommand
	Downloader         Downloader
	Unzzipper          Unzipper
}

type SeekerCredentials struct {
	SeekerServerURL string
}

func init() {
	logger := libbuildpack.NewLogger(os.Stdout)
	command := &libbuildpack.Command{}
	libbuildpack.AddHook(&SeekerAfterCompileHook{
		Log:        logger,
		Command:    command,
		Downloader: SeekerDownloader{},
		Unzzipper:  SeekerUnzipper{},
	})
}

func (h SeekerAfterCompileHook) AfterCompile(compiler *libbuildpack.Stager) error {
	h.Log.Debug("Seeker - AfterCompileHook Start")
	c := SeekerCredentials{}
	var err error
	if c, err = extractServiceCredentialsUserProvidedService(h.Log); err != nil {
		return err
	}
	if c == (SeekerCredentials{}) {
		if c, err = extractServiceCredentials(h.Log); err != nil {
			return err
		}
	}
	if err = assertServiceCredentialsValid(c); err != nil {
		return err
	}
	h.serviceCredentials = &c
	credentialsJSON, _ := json.Marshal(h.serviceCredentials)
	h.Log.Info("Credentials extraction ok: %s", credentialsJSON)

	if err = h.prependRequire(compiler); err != nil {
		return err
	}

	err, seekerLibraryToInstall := h.downloadAgent(compiler)
	if err != nil {
		return err
	}

	h.Log.Info("Before Installing seeker agent dependency")
	err = h.updateNodeModules(seekerLibraryToInstall, compiler.BuildDir())
	if err != nil {
		return err
	}
	h.Log.Info("After Installing seeker agent dependency")
	err = h.createSeekerEnvironmentScript(compiler)
	if err != nil {
		return errors.New("Error creating seeker-env.sh script: " + err.Error())
	}
	h.Log.Info("Done creating seeker-env.sh script")
	return nil
}

func (h SeekerAfterCompileHook) prependRequire(compiler *libbuildpack.Stager) error {
	entryPointPath := os.Getenv(EntryPointFile)
	if entryPointPath == "" {
		return nil
	}
	h.Log.Debug("Adding Seeker agent require to application entry point %s", entryPointPath)
	return h.addSeekerAgentRequire(compiler.BuildDir(), entryPointPath)
}

func (h SeekerAfterCompileHook) addSeekerAgentRequire(buildDir string, pathToEntryPointFile string) error {
	absolutePathToEntryPoint := filepath.Join(buildDir, pathToEntryPointFile)
	h.Log.Debug("Trying to prepend %s to %s", SeekerRequire, absolutePathToEntryPoint)
	c, err := ioutil.ReadFile(absolutePathToEntryPoint)
	if err != nil {
		return err
	}
	// do not require twice
	if pattern.Match(c) {
		return nil
	}
	return ioutil.WriteFile(absolutePathToEntryPoint, append([]byte(SeekerRequire), c...), 0644)
}

func assertServiceCredentialsValid(credentials SeekerCredentials) error {
	errorFormat := "mandatory `%s` is missing in Seeker service configuration"
	if credentials.SeekerServerURL == "" {
		return fmt.Errorf(errorFormat, "seeker_server_url")
	}
	return nil
}

func (h SeekerAfterCompileHook) downloadAgent(compiler *libbuildpack.Stager) (error, string) {
	parsedEnterpriseServerUrl, err := url.Parse(h.serviceCredentials.SeekerServerURL)
	if err != nil {
		return err, ""
	}
	parsedEnterpriseServerUrl.Path = path.Join(parsedEnterpriseServerUrl.Path, agentDownloadPath)
	agentDownloadAbsoluteUrl := parsedEnterpriseServerUrl.String()
	h.Log.Info("Agent download url %s", agentDownloadAbsoluteUrl)
	var seekerTempFolder = filepath.Join(os.TempDir(), "seeker_tmp")
	seekerLibraryPath := filepath.Join(os.TempDir(), "seeker-agent.tgz")
	os.RemoveAll(seekerTempFolder)
	os.Remove(seekerLibraryPath)
	err = os.MkdirAll(seekerTempFolder, 0755)
	if err != nil {
		return err, ""
	}
	agentZipAbsolutePath := path.Join(seekerTempFolder, "seeker-node-agent.zip")
	h.Log.Info("Downloading '%s' to '%s'", agentDownloadAbsoluteUrl, agentZipAbsolutePath)
	if err = h.Downloader.DownloadFile(agentDownloadAbsoluteUrl, agentZipAbsolutePath); err != nil {
		return err, ""
	}
	// no native zip support for unzip - using shell utility
	err = h.Unzzipper.Unzip(agentZipAbsolutePath, os.TempDir())
	if err != nil {
		return err, ""
	}
	if _, err := os.Stat(seekerLibraryPath); os.IsNotExist(err) {
		return errors.New("Could not find " + seekerLibraryPath), ""
	}
	// Cleanup unneeded files
	os.Remove(seekerTempFolder)
	return err, seekerLibraryPath
}
func (h SeekerAfterCompileHook) updateNodeModules(pathToSeekerLibrary string, buildDir string) error {
	// No need to handle YARN, since NPM is installed even when YARN is the selected package manager
	if err := h.Command.Execute(buildDir, ioutil.Discard, ioutil.Discard, "npm", "install", "--save", pathToSeekerLibrary, "--prefix", "seeker"); err != nil {
		h.Log.Error("npm install --save " + pathToSeekerLibrary + " --prefix seeker Error: " + err.Error())
		return err
	}
	return nil
}
func (h *SeekerAfterCompileHook) createSeekerEnvironmentScript(stager *libbuildpack.Stager) error {
	seekerEnvironmentScript := "seeker-env.sh"

	const seekerServerTemplate = "export SEEKER_SERVER_URL=%s\n"
	scriptContent := fmt.Sprintf(seekerServerTemplate, h.serviceCredentials.SeekerServerURL)
	stager.Logger().Info(seekerEnvironmentScript + " content: " + scriptContent)
	return stager.WriteProfileD(seekerEnvironmentScript, scriptContent)
}

func extractServiceCredentials(Log *libbuildpack.Logger) (SeekerCredentials, error) {
	type Service struct {
		Name         string `json:"name"`
		Label        string `json:"label"`
		InstanceName string `json:"instance_name"`
		BindingName  string `json:"binding_name"`
		Credentials  struct {
			EnterpriseServerUrl string `json:"enterprise_server_url"`
			SeekerServerUrl     string `json:"seeker_server_url"`
			SensorHost          string `json:"sensor_host"`
			SensorPort          string `json:"sensor_port"`
		} `json:"credentials"`
	}

	var vcapServices map[string][]Service

	err := json.Unmarshal([]byte(os.Getenv("VCAP_SERVICES")), &vcapServices)
	if err != nil {
		return SeekerCredentials{}, fmt.Errorf("failed to unmarshal VCAP_SERVICES: %s", err)
	}

	var detectedCredentials []SeekerCredentials

	for _, services := range vcapServices {
		for _, service := range services {
			if isSeekerRelated(service.Name, service.Label, service.InstanceName) {
				credentials := SeekerCredentials{SeekerServerURL: service.Credentials.SeekerServerUrl}
				detectedCredentials = append(detectedCredentials, credentials)
			}
		}
	}

	found, err := assertZeroOrOneServicesExist(len(detectedCredentials))
	if err != nil {
		return SeekerCredentials{}, err
	}
	if found {
		return detectedCredentials[0], nil
	}
	return SeekerCredentials{}, nil
}

func assertZeroOrOneServicesExist(c int) (bool, error) {
	if c > 1 {
		return false, fmt.Errorf("expected to find 1 Seeker service but found %d", c)
	}
	if c == 1 {
		return true, nil
	}

	return false, nil
}

func extractServiceCredentialsUserProvidedService(Log *libbuildpack.Logger) (SeekerCredentials, error) {
	type UserProvidedService struct {
		BindingName interface{} `json:"binding_name"`
		Credentials struct {
			SeekerServerUrl string `json:"seeker_server_url"`
		} `json:"credentials"`
		InstanceName   string   `json:"instance_name"`
		Label          string   `json:"label"`
		Name           string   `json:"name"`
		SyslogDrainURL string   `json:"syslog_drain_url"`
		Tags           []string `json:"tags"`
	}

	type VCAPSERVICES struct {
		UserProvidedService []UserProvidedService `json:"user-provided"`
	} //`json:"VCAP_SERVICES"`

	var vcapServices VCAPSERVICES
	vcapServicesString := os.Getenv("VCAP_SERVICES")
	if !strings.Contains(vcapServicesString, "user-provided") {
		return SeekerCredentials{}, nil
	}
	err := json.Unmarshal([]byte(vcapServicesString), &vcapServices)
	if err != nil {
		return SeekerCredentials{}, fmt.Errorf("failed to unmarshal VCAP_SERVICES: %s", err.Error())
	}

	var detectedCredentials []UserProvidedService

	for _, service := range vcapServices.UserProvidedService {
		if isSeekerRelated(service.Name, service.Label, service.InstanceName) {
			detectedCredentials = append(detectedCredentials, service)
		}
	}

	found, err := assertZeroOrOneServicesExist(len(detectedCredentials))
	if err != nil {
		return SeekerCredentials{}, err
	}
	if found {
		c := SeekerCredentials{
			SeekerServerURL: detectedCredentials[0].Credentials.SeekerServerUrl}
		return c, nil
	}
	return SeekerCredentials{}, nil
}

func isSeekerRelated(descriptors ...string) bool {
	isSeekerRelated := false
	for _, descriptor := range descriptors {
		containsSeeker, _ := regexp.MatchString(".*[sS][eE][eE][kK][eE][rR].*", descriptor)
		isSeekerRelated = isSeekerRelated || containsSeeker
	}
	return isSeekerRelated
}

type SeekerDownloader struct {
}

func (d SeekerDownloader) DownloadFile(url, destFile string) error {
	var err error
	var resp *http.Response
	if strings.HasPrefix(url, "https") {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		resp, err = http.Get(url)
	} else {
		resp, err = http.Get(url)
	}
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return errors.New("could not download: " + strconv.Itoa(resp.StatusCode))
	}
	return d.writeToFile(resp.Body, destFile, 0666)
}

func (d SeekerDownloader) writeToFile(source io.Reader, destFile string, mode os.FileMode) error {
	err := os.MkdirAll(filepath.Dir(destFile), 0755)
	if err != nil {
		return err
	}

	fh, err := os.OpenFile(destFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer fh.Close()

	_, err = io.Copy(fh, source)
	if err != nil {
		return err
	}
	return nil
}

type SeekerUnzipper struct {
}

func (s SeekerUnzipper) Unzip(zipFile, destFolder string) error {
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		p := filepath.Join(destFolder, f.Name)

		// Check for ZipSlip
		if !strings.HasPrefix(p, filepath.Clean(destFolder)+string(os.PathSeparator)) {
			return fmt.Errorf("%s: illegal file path", p)
		}

		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(p, os.ModePerm)
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(p), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}
		_, err = io.Copy(outFile, rc)

		if err = outFile.Close(); err != nil {
			return err
		}

		if err = rc.Close(); err != nil {
			return err
		}

	}
	return nil
}

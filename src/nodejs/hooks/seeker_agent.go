package hooks

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

const (
	SeekerRequire     = "require('/home/vcap/app/seeker/node_modules/@synopsys-sig/seeker');\n"
	entryPointFile    = "SEEKER_APP_ENTRY_POINT"
	agentDownloadPath = "/rest/api/latest/installers/agents/binaries/NODEJS"
	seekerTarGZ       = "seeker-agent.tgz"
	seekerZIP         = "seeker-node-agent.zip"
)

var pattern = regexp.MustCompile(`require\(['"].*@synopsys-sig/seeker['"]\)`)

type SeekerAfterCompileHook struct {
	libbuildpack.DefaultHook
	Log     *libbuildpack.Logger
	Command *libbuildpack.Command
}

type SeekerCredentials struct {
	ServiceName     string
	SeekerServerURL string
}

func init() {
	logger := libbuildpack.NewLogger(os.Stdout)
	command := &libbuildpack.Command{}
	libbuildpack.AddHook(&SeekerAfterCompileHook{
		Log:     logger,
		Command: command,
	})
}

func (h *SeekerAfterCompileHook) getAgentDownloadURL(c SeekerCredentials) (string, error) {
	agentDownloadUrl := os.Getenv("SEEKER_AGENT_DOWNLOAD_URL")
	if agentDownloadUrl != "" {
		h.Log.Debug("Using custom download URL: %s", agentDownloadUrl)
		return agentDownloadUrl, nil
	}

	parsedEnterpriseServerURL, err := url.Parse(c.SeekerServerURL)
	if err != nil {
		h.Log.Debug("Failed to parse SeekerServerURL")
		return "", err
	}
	parsedEnterpriseServerURL.Path = path.Join(parsedEnterpriseServerURL.Path, agentDownloadPath)
	return parsedEnterpriseServerURL.String(), nil
}

func (h *SeekerAfterCompileHook) AfterCompile(compiler *libbuildpack.Stager) error {
	h.Log.Debug("Seeker - AfterCompileHook Start")
	var err error
	c := h.getCredentials()
	if c == nil {
		h.Log.Debug("Seeker service credentials not found!")
		return nil
	}
	if err = h.PrependRequire(compiler); err != nil {
		return err
	}
	seekerTempFolder, err := os.MkdirTemp(os.TempDir(), "seeker_tmp")
	if err != nil {
		h.Log.Error("Failed to create temp dir")
		return err
	}
	tgzPath, err := h.downloadAgent(*c, seekerTempFolder)
	if err != nil {
		return err
	}

	appRoot := compiler.BuildDir()
	h.Log.Info("Before Installing seeker agent dependency to %s", appRoot)
	err = h.updateNodeModules(tgzPath, appRoot)
	if err != nil {
		return err
	}
	h.cleanupUnusedFiles(seekerTempFolder)
	h.Log.Info("After Installing seeker agent dependency")
	err = h.createSeekerEnvironmentScript(*c, compiler)
	if err != nil {
		return errors.New("Error creating seeker-env.sh script: " + err.Error())
	}
	h.Log.Info("Done creating seeker-env.sh script")
	return nil
}

func (h *SeekerAfterCompileHook) PrependRequire(compiler *libbuildpack.Stager) error {
	entryPointPath := os.Getenv(entryPointFile)
	if entryPointPath == "" {
		h.Log.Warning("%s is not defined, ignore this message if you required Seeker using: `require('/home/vcap/app/seeker/node_modules/@synopsys-sig/seeker')`", entryPointFile)
		return nil
	}
	absolutePathToEntryPoint := filepath.Join(compiler.BuildDir(), entryPointPath)
	c, err := os.ReadFile(absolutePathToEntryPoint)
	if err != nil {
		h.Log.Error("Failed to read entry point module: %s, Seeker agent will not be enabled", absolutePathToEntryPoint)
		return err
	}
	// do not require twice
	if pattern.Match(c) {
		h.Log.Debug("Seeker agent is already required...")
		return nil
	}
	h.Log.Debug("Trying to prepend %s to %s", SeekerRequire, absolutePathToEntryPoint)
	return os.WriteFile(absolutePathToEntryPoint, append([]byte(SeekerRequire), c...), 0644)
}

func (h *SeekerAfterCompileHook) downloadAgent(serviceCredentials SeekerCredentials, seekerTempFolder string) (string, error) {
	agentDownloadAbsoluteURL, err := h.getAgentDownloadURL(serviceCredentials)
	if err != nil {
		return "", err
	}

	h.Log.Info("Agent download url %s", agentDownloadAbsoluteURL)
	seekerLibraryPath := filepath.Join(seekerTempFolder, seekerTarGZ)
	agentZipAbsolutePath := path.Join(seekerTempFolder, seekerZIP)
	h.Log.Info("Downloading '%s' to '%s'", agentDownloadAbsoluteURL, agentZipAbsolutePath)
	if err = h.downloadFile(agentDownloadAbsoluteURL, agentZipAbsolutePath); err != nil {
		return "", err
	}
	err = libbuildpack.ExtractZip(agentZipAbsolutePath, seekerTempFolder)
	if err != nil {
		h.Log.Error("Failed to extract zip file: %s to folder: %s", agentZipAbsolutePath, seekerTempFolder)
		return "", err
	}

	exists, err := libbuildpack.FileExists(seekerLibraryPath)
	if !exists || err != nil {
		return "", errors.New("Could not find " + seekerLibraryPath)
	}
	return seekerLibraryPath, err
}

func (h *SeekerAfterCompileHook) updateNodeModules(tgzPath string, appRoot string) error {
	// No need to handle YARN, since NPM is installed even when YARN is the selected package manager
	h.Log.Debug("About to install seeker agent, build dir: %s, seeker package: %s", appRoot, tgzPath)
	var err error
	const seekerModule = "seeker"
	if os.Getenv("BP_DEBUG") != "" {
		err = h.Command.Execute(appRoot, os.Stdout, os.Stderr, "npm", "install", "--save", tgzPath, "--prefix", seekerModule)
	} else {
		err = h.Command.Execute(appRoot, io.Discard, io.Discard, "npm", "install", "--save", tgzPath, "--prefix", seekerModule)
	}
	if err != nil {
		h.Log.Error("npm install --save " + tgzPath + " --prefix seeker Error: " + err.Error())
		return err
	}
	seekerModuleDir := filepath.Join(appRoot, seekerModule)
	return h.listContents(seekerModuleDir)
}

func (h *SeekerAfterCompileHook) listContents(dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	h.Log.Debug("listing content of: %s", dir)
	for _, file := range files {
		h.Log.Debug(file.Name())
	}
	return nil
}

func (h *SeekerAfterCompileHook) createSeekerEnvironmentScript(serviceCredentials SeekerCredentials, stager *libbuildpack.Stager) error {
	seekerEnvironmentScript := "seeker-env.sh"

	const seekerServerTemplate = "export SEEKER_SERVER_URL=%s\n"
	scriptContent := fmt.Sprintf(seekerServerTemplate, serviceCredentials.SeekerServerURL)
	stager.Logger().Info(seekerEnvironmentScript + " content: " + scriptContent)
	return stager.WriteProfileD(seekerEnvironmentScript, scriptContent)
}

func (h *SeekerAfterCompileHook) getCredentials() *SeekerCredentials {
	var vcapServices map[string][]struct {
		Name        string                 `json:"name"`
		Credentials map[string]interface{} `json:"credentials"`
	}

	if err := json.Unmarshal([]byte(os.Getenv("VCAP_SERVICES")), &vcapServices); err != nil {
		h.Log.Debug("Failed to unmarshal VCAP_SERVICES: %s", err)
		return nil
	}

	var found []*SeekerCredentials

	for _, services := range vcapServices {
		for _, service := range services {
			if !strings.Contains(strings.ToLower(service.Name), "seeker") {
				continue
			}

			queryString := func(key string) string {
				if value, ok := service.Credentials[key].(string); ok {
					return value
				}
				return ""
			}

			c := &SeekerCredentials{
				ServiceName:     service.Name,
				SeekerServerURL: queryString("seeker_server_url"),
			}

			if c.SeekerServerURL == "" {
				h.Log.Warning("Empty value for Seeker Server URL in server: %s", c.ServiceName)
				continue
			}
			found = append(found, c)
		}
	}

	if len(found) == 1 {
		h.Log.Debug("Found one matching seeker service: %s", found[0].ServiceName)
		return found[0]
	}

	if len(found) > 1 {
		h.Log.Warning("More than one matching seeker service was found!")
	}

	return nil
}

func (h *SeekerAfterCompileHook) downloadFile(url, destFile string) error {
	resp, err := http.Get(url)
	if err != nil {
		h.Log.Error("Failed to download agent from: %s", url)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return errors.New("Download failed with HTTP status: " + strconv.Itoa(resp.StatusCode))
	}
	return h.writeToFile(resp.Body, destFile, 0666)
}

func (h *SeekerAfterCompileHook) writeToFile(source io.Reader, destFile string, mode os.FileMode) error {
	err := os.MkdirAll(filepath.Dir(destFile), 0755)
	if err != nil {
		h.Log.Error("Failed to create agent download directory %s", filepath.Dir(destFile))
		return err
	}

	fh, err := os.OpenFile(destFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		h.Log.Error("Failed to open file for writing (%s)", destFile)
		return err
	}
	defer fh.Close()

	_, err = io.Copy(fh, source)
	if err != nil {
		h.Log.Error("Failed to copy agent content to file (%s)", destFile)
		return err
	}
	return nil
}

func (h *SeekerAfterCompileHook) cleanupUnusedFiles(directory string) {
	files := [...]string{path.Join(directory, seekerTarGZ), path.Join(directory, seekerZIP)}
	for _, f := range files {
		os.Remove(f) // do not fail on error
	}
}

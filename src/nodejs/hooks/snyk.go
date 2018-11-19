package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

type SnykCommand interface {
	Output(string, string, ...string) (string, error)
}

// SnykHook
type SnykHook struct {
	libbuildpack.DefaultHook
	Log               *libbuildpack.Logger
	SnykCommand       SnykCommand
	buildDir          string
	depsDir           string
	localAgent        bool
	orgName           string
	severityThreshold string
}

type SnykCredentials struct {
	ApiToken string
	ApiUrl   string
	OrgName  string
}

const snykLocalAgentPath = "node_modules/snyk/cli/index.js"

func init() {
	logger := libbuildpack.NewLogger(os.Stdout)
	command := &libbuildpack.Command{}

	libbuildpack.AddHook(SnykHook{
		Log:               logger,
		SnykCommand:       command,
		buildDir:          "",
		depsDir:           "",
		localAgent:        true,
		orgName:           "",
		severityThreshold: "",
	})
}

//Snyk hook
func (h SnykHook) AfterCompile(stager *libbuildpack.Stager) error {
	if h.isTokenExists() == false {
		h.Log.Debug("Snyk token wasn't found...")
		return nil
	}
	h.Log.Debug("Snyk token was found.")
	h.Log.BeginStep("Checking if Snyk service is enabled...")

	dontBreakBuild := strings.ToLower(os.Getenv("SNYK_DONT_BREAK_BUILD")) == "true"
	monitorBuild := strings.ToLower(os.Getenv("SNYK_MONITOR_BUILD")) == "true"
	protectBuild := strings.ToLower(os.Getenv("SNYK_PROTECT_BUILD")) == "true"
	orgName := strings.ToLower(os.Getenv("SNYK_ORG_NAME"))
	severityThreshold := strings.ToLower(os.Getenv("SNYK_SEVERITY_THRESHOLD"))

	h.Log.Debug("SNYK_DONT_BREAK_BUILD is enabled: %t", dontBreakBuild)
	h.Log.Debug("SNYK_MONITOR_BUILD is enabled: %t", monitorBuild)
	h.Log.Debug("SNYK_PROTECT_BUILD is enabled: %t", protectBuild)
	if severityThreshold != "" {
		h.Log.Debug("SNYK_SEVERITY_THRESHOLD is set to: %s", severityThreshold)
	}

	h.buildDir = stager.BuildDir()
	h.depsDir = stager.DepDir()
	h.localAgent = true
	h.orgName = orgName
	h.severityThreshold = severityThreshold

	snykExists := h.isAgentExists()
	if snykExists == false {
		h.localAgent = false
		if err := h.installAgent(); err != nil {
			return err
		}
	}

	// make a temporary link to depsDir next to package.json, as this is what
	// snyk cli expects.
	depsDirLocalPath := filepath.Join(h.buildDir, "node_modules")
	depsDirGlobalPath := filepath.Join(h.depsDir, "node_modules")
	if _, err := os.Lstat(depsDirLocalPath); os.IsNotExist(err) {
		h.Log.Debug("%s does not exist. making a temporary symlink %s -> %s",
			depsDirLocalPath, depsDirLocalPath, depsDirGlobalPath)

		err := os.Symlink(depsDirGlobalPath, depsDirLocalPath)
		if err != nil {
			return err
		}

		defer func() {
			h.Log.Debug("removing temporary link %s", depsDirLocalPath)
			os.Remove(depsDirLocalPath)
		}()
	}

	if protectBuild {
		if err := h.runProtect(); err != nil {
			return err
		}
	}

	successfulRun, err := h.runTest()
	if err != nil {
		if !successfulRun {
			return err
		}

		if !dontBreakBuild {
			h.Log.Error("Snyk found vulnerabilties. Failing build...")
			return err
		}
		h.Log.Warning("SNYK_DONT_BREAK_BUILD was defined, continue build despite vulnerabilities found")
	}

	if monitorBuild {
		err = h.runMonitor()
		if err != nil {
			return err
		}
	}
	h.Log.Info("Snyk finished successfully")
	return nil
}

func (h SnykHook) isTokenExists() bool {
	token := os.Getenv("SNYK_TOKEN")
	if token != "" {
		return true
	}

	status, snykCredentials := h.getCredentialsFromService()
	if status {
		os.Setenv("SNYK_TOKEN", snykCredentials.ApiToken)
		if snykCredentials.ApiUrl != "" {
			os.Setenv("SNYK_API", snykCredentials.ApiUrl)
		}
		if snykCredentials.OrgName != "" {
			os.Setenv("SNYK_ORG_NAME", snykCredentials.OrgName)
		}
		return true
	}
	return false
}

func (h SnykHook) isAgentExists() bool {
	h.Log.Debug("Checking if Snyk agent exists...")
	snykCliPath := filepath.Join(h.buildDir, snykLocalAgentPath)
	if _, err := os.Stat(snykCliPath); os.IsNotExist(err) {
		h.Log.Debug("Snyk agent doesn't exist")
		return false
	}

	h.Log.Debug("Snyk agent exists")
	return true
}

func (h SnykHook) installAgent() error {
	h.Log.Info("Installing Snyk agent...")
	output, err := h.SnykCommand.Output(h.buildDir, "npm", "install", "-g", "snyk")
	if err == nil {
		h.Log.Debug("Snyk agent installed %s", output)
		return nil
	}
	h.Log.Warning("Failed to install Snyk agent, please add snyk to your package.json dependecies.")
	return err
}

func (h SnykHook) runSnykCommand(args ...string) (string, error) {
	if h.orgName != "" {
		args = append(args, "--org="+h.orgName)
	}

	if os.Getenv("BP_DEBUG") != "" {
		args = append(args, "-d")
	}

	if h.severityThreshold != "" {
		args = append(args, fmt.Sprintf("--severity-threshold=%s", h.severityThreshold))
	}

	// Snyk is part of the app modules.
	if h.localAgent == true {
		snykCliPath := filepath.Join(h.buildDir, snykLocalAgentPath)
		snykArgs := append([]string{snykCliPath}, args...)
		return h.SnykCommand.Output(h.buildDir, "node", snykArgs...)
	}

	// Snyk is installed globally.
	snykGlobalAgentPath := filepath.Join(h.depsDir, "node", "bin", "snyk")
	return h.SnykCommand.Output(h.buildDir, snykGlobalAgentPath, args...)
}

func (h SnykHook) runTest() (bool, error) {
	h.Log.Debug("Run Snyk test...")
	output, err := h.runSnykCommand("test")
	if err == nil {
		h.Log.Info("Snyk test finished successfully - %s", output)
		return true, nil
	}
	//In case we got an unexpected output.
	if !strings.Contains(output, "dependencies for known") {
		h.Log.Warning("Failed to run Snyk agent - %s", output)
		h.Log.Warning("Please validate your auth token and that your npm version is equal or greater than v3.x.x")
		return false, err
	}
	h.Log.Warning("Snyk found vulnerabilties - %s", output)
	return true, err
}

func (h SnykHook) runMonitor() error {
	h.Log.Debug("Run Snyk monitor...")
	output, err := h.runSnykCommand("monitor", "--project-name="+h.appName())
	h.Log.Info("Snyk monitor %s", output)
	return err
}

func (h SnykHook) isPolicyFileExists() bool {
	h.Log.Debug("Check for Snyk policy file...")
	policyFilePath := filepath.Join(h.buildDir, ".snyk")
	if _, err := os.Stat(policyFilePath); os.IsNotExist(err) {
		return false
	}
	return true
}

func (h SnykHook) runProtect() error {
	if !h.isPolicyFileExists() {
		return nil
	}

	_, err := h.runSnykCommand("protect")
	return err
}

func getCredentialString(credentials map[string]interface{}, key string) string {
	value, isString := credentials[key].(string)

	if isString {
		return value
	}
	return ""
}

func (h SnykHook) getCredentialsFromService() (bool, SnykCredentials) {
	type Service struct {
		Name        string                 `json:"name"`
		Credentials map[string]interface{} `json:"credentials"`
	}
	var vcapServices map[string][]Service
	err := json.Unmarshal([]byte(os.Getenv("VCAP_SERVICES")), &vcapServices)
	if err != nil {
		h.Log.Warning("Failed to parse VCAP_SERVICES")
		return false, SnykCredentials{}
	}

	for key, services := range vcapServices {
		if strings.Contains(key, "snyk") {
			for _, service := range services {
				apiToken := getCredentialString(service.Credentials, "apiToken")
				if apiToken != "" {
					apiUrl := getCredentialString(service.Credentials, "apiUrl")
					orgName := getCredentialString(service.Credentials, "orgName")
					snykCredantials := SnykCredentials{
						ApiToken: apiToken,
						ApiUrl:   apiUrl,
						OrgName:  orgName,
					}
					return true, snykCredantials
				}
			}
		}
	}

	return false, SnykCredentials{}
}

func (h SnykHook) appName() string {
	var application struct {
		Name string `json:"name"`
	}
	err := json.Unmarshal([]byte(os.Getenv("VCAP_APPLICATION")), &application)
	if err != nil {
		return ""
	}

	return application.Name
}

package hooks

import (
	"encoding/json"
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
	Log         *libbuildpack.Logger
	SnykCommand SnykCommand
	buildDir    string
	depsDir     string
	localAgent  bool
}

const snykLocalAgentPath = "node_modules/snyk/cli/index.js"

func init() {
	logger := libbuildpack.NewLogger(os.Stdout)
	command := &libbuildpack.Command{}

	libbuildpack.AddHook(SnykHook{
		Log:         logger,
		SnykCommand: command,
		buildDir:    "",
		depsDir:     "",
		localAgent:  true,
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

	ignoreVulns := strings.ToLower(os.Getenv("SNYK_IGNORE_VULNS")) == "true"
	monitorBuild := strings.ToLower(os.Getenv("SNYK_MONITOR_BUILD")) == "true"
	protectBuild := strings.ToLower(os.Getenv("SNYK_PROTECT_BUILD")) == "true"

	h.Log.Debug("SNYK_IGNORE_VULNS is enabled: %t", ignoreVulns)
	h.Log.Debug("SNYK_MONITOR_BUILD is enabled: %t", monitorBuild)
	h.Log.Debug("SNYK_PROTECT_BUILD is enabled: %t", protectBuild)

	h.buildDir = stager.BuildDir()
	h.depsDir = stager.DepDir()
	h.localAgent = true

	snykExists := h.isAgentExists()
	if snykExists == false {
		h.localAgent = false
		if err := h.installAgent(); err != nil {
			return err
		}
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

		if !ignoreVulns {
			h.Log.Error("Snyk found vulnerabilties. Failing build...")
			return err
		}
		h.Log.Warning("SNYK_IGNORE_VULNS was defined, continue build despite vulnerabilities found")
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

	token = h.getTokenFromCredentials()
	if token != "" {
		os.Setenv("SNYK_TOKEN", token)
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
	output, err := h.runSnykCommand("test", "-d")
	if err == nil {
		return true, nil
	}
	//In case we got an unexpected output.
	if !strings.Contains(output, "dependencies for known issues") {
		h.Log.Warning("Failed to run Snyk agent - %s", output)
		h.Log.Warning("Please validate your auth token and that your npm version is equal or greater than v3.x.x")
		return false, err
	}
	return true, err
}

func (h SnykHook) runMonitor() error {
	h.Log.Debug("Run Snyk monitor...")
	output, err := h.runSnykCommand("monitor", "--project-name="+h.appName())
	h.Log.Debug("Snyk monitor %s", output)
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

func (h SnykHook) getTokenFromCredentials() string {
	type Service struct {
		Name        string                 `json:"name"`
		Credentials map[string]interface{} `json:"credentials"`
	}
	var vcapServices map[string][]Service
	err := json.Unmarshal([]byte(os.Getenv("VCAP_SERVICES")), &vcapServices)
	if err != nil {
		h.Log.Warning("Failed to parse VCAP_SERVICES")
		return ""
	}

	for _, services := range vcapServices {
		for _, service := range services {
			if strings.Contains(service.Name, "snyk") {
				if serviceToken, ok := service.Credentials["SNYK_TOKEN"]; ok {
					return serviceToken.(string)
				}
			}
		}
	}

	return ""
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

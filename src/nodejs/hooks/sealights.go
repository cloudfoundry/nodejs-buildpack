package hooks

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

const EmptyTokenError = "token cannot be empty (env SL_TOKEN | SL_TOKEN_FILE)"
const CommandStringError = "Cannot find command begin term"
const NpmCommandStringError = "NPM command without package.json file is not supported"
const SealightsNotBoundError = "Sealights service not bound"
const EmptyBuildError = "build session id cannot be empty (env SL_BUILD_SESSION_ID | SL_BUILD_SESSION_ID_FILE)"
const Procfile = "Procfile"
const PackageJsonFile = "package.json"
const ManifestFile = "manifest.yml"

type Command interface {
	Execute(dir string, stdout io.Writer, stderr io.Writer, program string, args ...string) error
}

type SealightsHook struct {
	libbuildpack.DefaultHook
	Log     *libbuildpack.Logger
	Command Command
}

type SealightsOptions struct {
	Token       string
	TokenFile   string
	BsId        string
	BsIdFile    string
	Proxy       string
	LabId       string
	ProjectRoot string
	TestStage   string
	App         string
}

type Manifest struct {
	Applications []struct {
		Name    string `yaml:"name"`
		Command string `yaml:"command"`
	} `yaml:"applications"`
}

type PackageJson struct {
	Name    string `json:"name"`
	Command string `json:"main"`
}

func init() {
	logger := libbuildpack.NewLogger(os.Stdout)
	command := &libbuildpack.Command{}
	libbuildpack.AddHook(&SealightsHook{
		Log:     logger,
		Command: command,
	})
}

func (sl *SealightsHook) AfterCompile(stager *libbuildpack.Stager) error {
	sl.Log.Info("inside Sealights hook")

	if !sl.RunWithSealights() {
		sl.Log.Info("service is not configured to run with Sealights")
		return nil
	}

	err := sl.injectSealights(stager)
	if err != nil {
		sl.Log.Error("error injecting Sealights: %s", err)
		return nil
	}

	err = sl.installAgent(stager)
	if err != nil {
		return err
	}

	return nil
}

func (sl *SealightsHook) RunWithSealights() bool {
	isTokenFound, _, _ := sl.GetTokenFromEnvironment()
	return isTokenFound
}

func (sl *SealightsHook) SetApplicationStartInProcfile(stager *libbuildpack.Stager) error {
	bytes, err := os.ReadFile(filepath.Join(stager.BuildDir(), Procfile))
	if err != nil {
		sl.Log.Error("failed to read %s", Procfile)
		return err
	}

	originalStartCommand := string(bytes)
	_, usePackageJson := sl.usePackageJson(originalStartCommand, stager)
	if usePackageJson {
		// move to package json scenario
		return sl.SetApplicationStartInPackageJson(stager)
	}

	// we suppose that format is "web: node <application>"
	var newCmd string
	err, newCmd = sl.updateStartCommand(originalStartCommand)

	if err != nil {
		return err
	}

	if newCmd == "" {
		return nil
	}

	startCommand := "web: " + newCmd

	err = os.WriteFile(filepath.Join(stager.BuildDir(), Procfile), []byte(startCommand), 0755)
	if err != nil {
		sl.Log.Error("failed to update %s, error: %s", Procfile, err.Error())
		return err
	}

	return nil
}

func (sl *SealightsHook) usePackageJson(originalStartCommand string, stager *libbuildpack.Stager) (error, bool) {

	isNpmCommand, err := regexp.MatchString(`(^(web:\s)?cd[^&]*\s&&\snpm)|(^(web:\s)?npm)`, originalStartCommand)
	if err != nil {
		return err, false
	}

	isPackageExists := fileExists(filepath.Join(stager.BuildDir(), PackageJsonFile))
	if !isNpmCommand {
		return err, false
	}

	if isNpmCommand && isPackageExists {
		// move to package json scenario
		return nil, true
	}

	// case with npm command without package.json is not supported
	return fmt.Errorf(NpmCommandStringError), false
}

func (sl *SealightsHook) getSealightsOptions(app string, token string, tokenFile string) *SealightsOptions {
	o := &SealightsOptions{
		Token:       token,
		TokenFile:   tokenFile,
		BsId:        os.Getenv("SL_BUILD_SESSION_ID"),
		BsIdFile:    os.Getenv("SL_BUILD_SESSION_ID_FILE"),
		Proxy:       os.Getenv("SL_PROXY"),
		LabId:       os.Getenv("SL_LAB_ID"),
		ProjectRoot: os.Getenv("SL_PROJECT_ROOT"),
		TestStage:   os.Getenv("SL_TEST_STAGE"),
		App:         app,
	}
	return o
}

func (sl *SealightsHook) SetApplicationStartInPackageJson(stager *libbuildpack.Stager) error {
	packageJson, err := sl.ReadPackageJson(stager)
	if err != nil {
		return err
	}
	scripts, _ := packageJson["scripts"].(map[string]interface{})
	if scripts == nil {
		return fmt.Errorf("failed to read scripts from %s", PackageJsonFile)
	}
	originalStartScript, _ := scripts["start"].(string)
	if originalStartScript == "" {
		return fmt.Errorf("failed to read start from scripts in %s", PackageJsonFile)
	}
	// we suppose that format is "start: node <application>"
	var newCmd string
	err, newCmd = sl.updateStartCommand(originalStartScript)
	if err != nil {
		return err
	}
	packageJson["scripts"].(map[string]interface{})["start"] = newCmd

	err = libbuildpack.NewJSON().Write(filepath.Join(stager.BuildDir(), PackageJsonFile), packageJson)
	if err != nil {
		sl.Log.Error("failed to update %s, error: %s", PackageJsonFile, err.Error())
		return err
	}

	return nil
}

func (sl *SealightsHook) ReadPackageJson(stager *libbuildpack.Stager) (map[string]interface{}, error) {
	p := map[string]interface{}{}

	if err := libbuildpack.NewJSON().Load(filepath.Join(stager.BuildDir(), "package.json"), &p); err != nil {
		if err != nil {
			sl.Log.Error("failed to read %s error: %s", Procfile, err.Error())
			return nil, err
		}
	}
	return p, nil
}

func (sl *SealightsHook) SetApplicationStartInManifest(stager *libbuildpack.Stager) error {
	y := &libbuildpack.YAML{}
	err, m := sl.ReadManifestFile(stager, y)
	if err != nil {
		return err
	}
	originalStartCommand := m.Applications[0].Command

	_, usePackageJson := sl.usePackageJson(originalStartCommand, stager)
	if usePackageJson {
		// move to package json scenario
		return sl.SetApplicationStartInPackageJson(stager)
	}

	// we suppose that format is "start: node <application>"
	var newCmd string
	err, newCmd = sl.updateStartCommand(originalStartCommand)
	if err != nil {
		return err
	}

	m.Applications[0].Command = newCmd
	err = y.Write(filepath.Join(stager.BuildDir(), ManifestFile), m)
	if err != nil {
		sl.Log.Error("failed to update %s, error: %s", ManifestFile, err.Error())
		return err
	}

	return nil
}

func (sl *SealightsHook) updateStartCommand(originalCommand string) (error, string) {
	slTokenFound, token, tokenFile := sl.GetTokenFromEnvironment()

	if !slTokenFound {
		sl.Log.Info("Sealights service not found")
		return fmt.Errorf(SealightsNotBoundError), ""
	}

	split := strings.SplitAfterN(originalCommand, "node", 2)

	if len(split) < 2 {
		return fmt.Errorf(CommandStringError), ""
	}
	o := sl.getSealightsOptions(split[1], token, tokenFile)

	err := sl.validate(o)
	if err != nil {
		return err, ""
	}
	newCmd := sl.createAppStartCommandLine(o)
	sl.Log.Debug("new start script: %s", newCmd)
	return nil, newCmd
}

func (sl *SealightsHook) ReadManifestFile(stager *libbuildpack.Stager, y *libbuildpack.YAML) (error, Manifest) {
	var m Manifest
	if err := y.Load(filepath.Join(stager.BuildDir(), ManifestFile), &m); err != nil {
		if err != nil {
			sl.Log.Error("failed to read %s error: %s", ManifestFile, err.Error())
			return err, m
		}
	}
	return nil, m
}

func (sl *SealightsHook) installAgent(stager *libbuildpack.Stager) error {
	err := sl.Command.Execute(stager.BuildDir(), os.Stdout, os.Stderr, "npm", "install", "slnodejs")
	if err != nil {
		sl.Log.Error("npm install slnodejs failed with error: " + err.Error())
		return err
	}
	sl.Log.Info("npm install slnodejs finished successfully")
	return nil
}

func (sl *SealightsHook) createAppStartCommandLine(o *SealightsOptions) string {
	var sb strings.Builder
	sb.WriteString("./node_modules/.bin/slnodejs run  --useinitialcolor true ")

	if o.TokenFile != "" {
		sb.WriteString(fmt.Sprintf(" --tokenfile %s", o.TokenFile))
	} else {
		sb.WriteString(fmt.Sprintf(" --token %s", o.Token))
	}

	if o.BsIdFile != "" {
		sb.WriteString(fmt.Sprintf(" --buildsessionidfile %s", o.BsIdFile))
	} else {
		sb.WriteString(fmt.Sprintf(" --buildsessionid %s", o.BsId))
	}

	if o.Proxy != "" {
		sb.WriteString(fmt.Sprintf(" --proxy %s ", o.Proxy))
	}

	if o.LabId != "" {
		sb.WriteString(fmt.Sprintf(" --labid %s ", o.LabId))
	}

	if o.ProjectRoot != "" {
		sb.WriteString(fmt.Sprintf(" --projectroot %s ", o.ProjectRoot))
	}

	// test stage contains white space(e.g. "Unit Tests", make it quoted
	if o.TestStage != "" {
		sb.WriteString(fmt.Sprintf(" --teststage \"%s\" ", o.TestStage))
	}

	sb.WriteString(fmt.Sprintf(" %s", o.App))
	return sb.String()
}

func (sl *SealightsHook) validate(o *SealightsOptions) error {
	if o.Token == "" && o.TokenFile == "" {
		sl.Log.Error(EmptyTokenError)
		return fmt.Errorf(EmptyTokenError)
	}

	if o.BsId == "" && o.BsIdFile == "" {
		sl.Log.Error(EmptyBuildError)
		return fmt.Errorf(EmptyBuildError)
	}

	return nil
}

func (sl *SealightsHook) injectSealights(stager *libbuildpack.Stager) error {
	if _, err := os.Stat(filepath.Join(stager.BuildDir(), Procfile)); err == nil {
		sl.Log.Info("Integrating sealights into procfile")
		return sl.SetApplicationStartInProcfile(stager)
	} else if _, err := os.Stat(filepath.Join(stager.BuildDir(), ManifestFile)); err == nil {
		sl.Log.Info("Integrating sealights into manifest.yml")
		return sl.SetApplicationStartInManifest(stager)
	} else {
		sl.Log.Info("Integrating sealights into package.json")
		return sl.SetApplicationStartInPackageJson(stager)
	}
}

func containsSealightsService(key string, services interface{}, query string) bool {
	var serviceName string

	if strings.Contains(key, query) {
		return true
	}
	val := services.([]interface{})
	for serviceIndex := range val {
		service := val[serviceIndex].(map[string]interface{})
		if v, ok := service["name"]; ok {
			serviceName = v.(string)
		}
		if strings.Contains(serviceName, query) {
			return true
		}
	}
	return false
}

func (sl *SealightsHook) GetTokenFromEnvironment() (bool, string, string) {

	type rawVcapServicesJSONValue map[string]interface{}

	var vcapServices rawVcapServicesJSONValue

	vcapServicesEnvironment := os.Getenv("VCAP_SERVICES")

	if vcapServicesEnvironment == "" {
		sl.Log.Debug("Sealights could not find VCAP_SERVICES in the environment")
		return false, "", ""
	}

	err := json.Unmarshal([]byte(vcapServicesEnvironment), &vcapServices)
	if err != nil {
		sl.Log.Warning("Sealights could not parse VCAP_SERVICES")
		return false, "", ""
	}

	for key, services := range vcapServices {
		if containsSealightsService(key, services, "sealights") {
			sl.Log.Debug("Sealights found credentials in VCAP_SERVICES")
			val := services.([]interface{})
			for serviceIndex := range val {
				service := val[serviceIndex].(map[string]interface{})
				if credentials, exists := service["credentials"].(map[string]interface{}); exists {
					token := getContrastCredentialString(credentials, "token")
					tokenFile := getContrastCredentialString(credentials, "tokenFile")
					if token == "" && tokenFile == "" {
						return false, "", ""
					}
					return true, token, tokenFile
				}
			}
		}
	}
	return false, "", ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

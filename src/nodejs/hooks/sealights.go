package hooks

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

const EmptyTokenError = "token cannot be empty (env SL_TOKEN | SL_TOKEN_FILE)"
const CommandStringError = "cannot find command begin term"
const NpmCommandStringError = "nmp command without package.json file is not supported"
const SealightsNotBoundError = "sealights service not bound"
const EmptyBuildError = "build session id cannot be empty (env SL_BUILD_SESSION_ID | SL_BUILD_SESSION_ID_FILE)"
const Procfile = "Procfile"
const PackageJsonFile = "package.json"
const ManifestFile = "manifest.yml"
const DefaultVersion = "latest"
const DefaultPackage = "slnodejs"
const AgentPackageVersionFormat = "%s@%s"
const AgentRecommendedVersionUrlFormat = "https://%s.sealights.co/api/v2/agents/slnodejs/recommended"

type Command interface {
	Execute(dir string, stdout io.Writer, stderr io.Writer, program string, args ...string) error
}

// HTTPClient interface represents the basic HTTP client operations.
type HttpClient interface {
	Get(url string) (*http.Response, error)
}

type SealightsHook struct {
	libbuildpack.DefaultHook
	Log        *libbuildpack.Logger
	Command    Command
	HttpClient HttpClient

	parameters  *SealightsParameters
	initialized bool
}

type SealightsParameters struct {
	Token              string
	TokenFile          string
	BuildSessionId     string
	BuildSessionIdFile string
	LabId              string
	CustomAgentUrl     string
	Version            string
	Proxy              string
	ProxyUsername      string
	ProxyPassword      string
	ProjectRoot        string
	TestStage          string
}

type SealightsRunOptions struct {
	Token              string
	TokenFile          string
	BuildSessionId     string
	BuildSessionIdFile string
	Proxy              string
	ProxyUsername      string
	ProxyPassword      string
	LabId              string
	ProjectRoot        string
	TestStage          string // depracated
	App                string
}

type RecomendedVersionResponse struct {
	Type  string       `json:"type"`
	Agent AgentVersion `json:"agent"`
}

type AgentVersion struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Date    string `json:"date"`
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
	hook := NewSealightsHook(logger, command, nil)
	libbuildpack.AddHook(hook)
}

func NewSealightsHook(logger *libbuildpack.Logger, command Command, httpClient HttpClient) *SealightsHook {
	return &SealightsHook{
		DefaultHook: libbuildpack.DefaultHook{},
		Log:         logger,
		Command:     command,
		HttpClient:  httpClient,
		parameters:  &SealightsParameters{},
		initialized: false,
	}
}

func (sl *SealightsHook) AfterCompile(stager *libbuildpack.Stager) error {
	sl.parseVcapServices()

	if !sl.RunWithSealights() {
		sl.Log.Debug("service is not configured to run with Sealights")
		return nil
	}

	err := sl.injectSealights(stager)
	if err != nil {
		sl.Log.Error("error injecting Sealights: %s", err)
		return nil
	}

	err = sl.installAgent(stager)
	if err != nil {
		sl.Log.Error("error injecting Sealights: %s", err)
		return err
	}

	return nil
}

func (sl *SealightsHook) RunWithSealights() bool {
	sl.parseVcapServices()
	return sl.parameters.Token != "" || sl.parameters.TokenFile != ""
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
	newCmd, err = sl.updateStartCommand(originalStartCommand)

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

func (sl *SealightsHook) getSealightsOptions(app string) *SealightsRunOptions {

	buildSessionId := os.Getenv("SL_BUILD_SESSION_ID")
	if sl.parameters.BuildSessionId != "" {
		buildSessionId = sl.parameters.BuildSessionId
	}

	buildSessionIdFile := os.Getenv("SL_BUILD_SESSION_ID_FILE")
	if sl.parameters.BuildSessionIdFile != "" {
		buildSessionIdFile = sl.parameters.BuildSessionIdFile
	}

	proxy := os.Getenv("SL_PROXY")
	if sl.parameters.Proxy != "" {
		proxy = sl.parameters.Proxy
	}

	labId := os.Getenv("SL_LAB_ID")
	if sl.parameters.LabId != "" {
		labId = sl.parameters.LabId
	}

	projectRoot := os.Getenv("SL_PROJECT_ROOT")
	if sl.parameters.ProjectRoot != "" {
		projectRoot = sl.parameters.ProjectRoot
	}

	testStage := os.Getenv("SL_TEST_STAGE")
	if sl.parameters.ProjectRoot != "" {
		testStage = sl.parameters.TestStage
	}

	o := &SealightsRunOptions{
		Token:              sl.parameters.Token,
		TokenFile:          sl.parameters.TokenFile,
		BuildSessionId:     buildSessionId,
		BuildSessionIdFile: buildSessionIdFile,
		Proxy:              proxy,
		ProxyUsername:      sl.parameters.ProxyUsername,
		ProxyPassword:      sl.parameters.ProxyPassword,
		LabId:              labId,
		ProjectRoot:        projectRoot,
		TestStage:          testStage,
		App:                app,
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
	newCmd, err = sl.updateStartCommand(originalStartScript)
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
	newCmd, err = sl.updateStartCommand(originalStartCommand)
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

func (sl *SealightsHook) updateStartCommand(originalCommand string) (string, error) {

	if !sl.RunWithSealights() {
		sl.Log.Info("Sealights service not found")
		return "", fmt.Errorf(SealightsNotBoundError)
	}

	split := strings.SplitAfterN(originalCommand, "node", 2)

	if len(split) < 2 {
		return "", fmt.Errorf(CommandStringError)
	}
	o := sl.getSealightsOptions(split[1])

	err := sl.validate(o)
	if err != nil {
		return "", err
	}
	newCmd := sl.createAppStartCommandLine(o)
	sl.Log.Debug("new start script: %s", newCmd)
	return newCmd, nil
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
	packageName, source := sl.getPackageName()
	sl.Log.Info("npm install %s\nversion source: %s", packageName, source)
	err := sl.Command.Execute(stager.BuildDir(), os.Stdout, os.Stderr, "npm", "install", packageName)
	if err != nil {
		sl.Log.Error("npm install %s failed with error: %s", packageName, err.Error())
		return err
	}
	sl.Log.Info("npm install %s finished successfully", packageName)
	return nil
}

func (sl *SealightsHook) createAppStartCommandLine(o *SealightsRunOptions) string {
	var sb strings.Builder
	sb.WriteString("./node_modules/.bin/slnodejs run  --useinitialcolor true ")

	if o.TokenFile != "" {
		sb.WriteString(fmt.Sprintf(" --tokenfile %s", o.TokenFile))
	} else {
		sb.WriteString(fmt.Sprintf(" --token %s", o.Token))
	}

	if o.BuildSessionIdFile != "" {
		sb.WriteString(fmt.Sprintf(" --buildsessionidfile %s", o.BuildSessionIdFile))
	} else {
		sb.WriteString(fmt.Sprintf(" --buildsessionid %s", o.BuildSessionId))
	}

	if o.Proxy != "" {
		sb.WriteString(fmt.Sprintf(" --proxy %s ", o.Proxy))
	}

	if o.ProxyUsername != "" {
		sb.WriteString(fmt.Sprintf(" --proxyUsername %s ", o.ProxyUsername))
	}

	if o.ProxyPassword != "" {
		sb.WriteString(fmt.Sprintf(" --proxyPassword %s ", o.ProxyPassword))
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

func (sl *SealightsHook) validate(o *SealightsRunOptions) error {
	if o.Token == "" && o.TokenFile == "" {
		sl.Log.Error(EmptyTokenError)
		return fmt.Errorf(EmptyTokenError)
	}

	if o.BuildSessionId == "" && o.BuildSessionIdFile == "" {
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

func (sl *SealightsHook) parseVcapServices() {

	if sl.initialized {
		sl.Log.Debug("already initialized. config won`t be parsed")
		return
	} else {
		sl.initialized = true
	}

	var vcapServices map[string][]struct {
		Name        string                 `json:"name"`
		Credentials map[string]interface{} `json:"credentials"`
	}

	if err := json.Unmarshal([]byte(os.Getenv("VCAP_SERVICES")), &vcapServices); err != nil {
		sl.Log.Debug("Failed to unmarshal VCAP_SERVICES: %s", err)
		return
	}

	for _, services := range vcapServices {
		for _, service := range services {
			if !strings.Contains(strings.ToLower(service.Name), "sealights") {
				continue
			}

			queryString := func(key string) string {
				if value, ok := service.Credentials[key].(string); ok {
					return value
				}
				return ""
			}

			options := &SealightsParameters{
				Token:              queryString("token"),
				TokenFile:          queryString("tokenFile"),
				BuildSessionId:     queryString("buildSessionId"),
				BuildSessionIdFile: queryString("buildSessionIdFile"),
				LabId:              queryString("labId"),
				Version:            queryString("version"),
				CustomAgentUrl:     queryString("customAgentUrl"),
				Proxy:              queryString("proxy"),
				ProxyUsername:      queryString("proxyUsername"),
				ProxyPassword:      queryString("proxyPassword"),
				ProjectRoot:        queryString("projectRoot"),
				TestStage:          queryString("testStage"),
			}

			// write warning in case token is not provided
			if options.Token != "" && options.TokenFile != "" {
				sl.Log.Warning("Sealights access token isn't provided")
			}

			sl.parameters = options
			return
		}
	}

}

func (sl *SealightsHook) getPackageName() (string, string) {
	if sl.parameters.CustomAgentUrl != "" {
		return sl.parameters.CustomAgentUrl, "customAgentUrl parameter"
	}

	source := "DefaultVersion"
	version := DefaultVersion
	if sl.parameters.Version != "" {
		version = sl.parameters.Version
		source = "version parameter"
		return fmt.Sprintf(AgentPackageVersionFormat, DefaultPackage, version), source
	}

	recomendedVersion, err := sl.getRecomendedAgentVersionFromServer()
	if err != nil {
		sl.Log.Warning(err.Error())
	} else {
		version = recomendedVersion
		source = "recomended version from server"
	}

	return fmt.Sprintf(AgentPackageVersionFormat, DefaultPackage, version), source
}

func (sl *SealightsHook) getRecomendedAgentVersionFromServer() (string, error) {
	domain := os.Getenv("SL_DOMAIN")
	if domain == "" {
		return "", errors.New("env variable \"SL_DOMAIN\" is not defined. recomended version wouldn't be requested")
	}

	url := fmt.Sprintf(AgentRecommendedVersionUrlFormat, domain)

	client := sl.createHttpClient()

	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Parse the JSON response
	var response RecomendedVersionResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}

	// Return the version from the parsed response
	return response.Agent.Version, nil
}

// Create simple http client or http client with proxy, based on the settings
func (sl *SealightsHook) createHttpClient() HttpClient {
	if sl.HttpClient != nil {
		return sl.HttpClient
	}

	if sl.parameters.Proxy != "" {
		proxyUrl, _ := url.Parse(sl.parameters.Proxy)

		sl.HttpClient = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(&url.URL{
					Scheme: proxyUrl.Scheme,
					User:   url.UserPassword(sl.parameters.ProxyUsername, sl.parameters.ProxyPassword),
					Host:   proxyUrl.Host,
				}),
			},
		}
		return sl.HttpClient
	} else {
		sl.HttpClient = &http.Client{}
		return sl.HttpClient
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

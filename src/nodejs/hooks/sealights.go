package hooks

import (
	"encoding/json"
	"fmt"
	"github.com/cloudfoundry/libbuildpack"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const EmptyTokenError = "token cannot be empty (env SL_TOKEN | SL_TOKEN_FILE)"
const EmptyBuildError = "build session id cannot be empty (env SL_BUILD_SESSION_ID | SL_BUILD_SESSION_ID_FILE)"
const Procfile = "Procfile"

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

func init() {
	logger := libbuildpack.NewLogger(os.Stdout)
	command := &libbuildpack.Command{}
	libbuildpack.AddHook(&SealightsHook{
		Log:     logger,
		Command: command,
	})
}

func (sl *SealightsHook) AfterCompile(stager *libbuildpack.Stager) error {
	if !sl.isSealightsBound() {
		return nil
	}

	sl.Log.Info("Inside Sealights hook")

	err := sl.SetApplicationStart(stager)
	if err != nil {
		return err
	}

	err = sl.installAgent(stager)
	if err != nil {
		return err
	}

	return nil
}

func (sl *SealightsHook) SetApplicationStart(stager *libbuildpack.Stager) error {
	bytes, err := ioutil.ReadFile(filepath.Join(stager.BuildDir(), Procfile))
	if err != nil {
		sl.Log.Error("failed to read %s", Procfile)
		return err
	}

	// we suppose that format is "web: node <application>"
	split := strings.SplitAfter(string(bytes), "node")

	o := &SealightsOptions{
		Token:       os.Getenv("SL_TOKEN"),
		TokenFile:   os.Getenv("SL_TOKEN_FILE"),
		BsId:        os.Getenv("SL_BUILD_SESSION_ID"),
		BsIdFile:    os.Getenv("SL_BUILD_SESSION_ID_FILE"),
		Proxy:       os.Getenv("SL_PROXY"),
		LabId:       os.Getenv("SL_LAB_ID"),
		ProjectRoot: os.Getenv("SL_PROJECT_ROOT"),
		TestStage:   os.Getenv("SL_TEST_STAGE"),
		App:         split[1],
	}

	err = sl.validate(o)
	if err != nil {
		return err
	}

	newCmd := sl.createAppStartCommandLine(o)

	sl.Log.Debug("new command line: %s", newCmd)

	err = ioutil.WriteFile(filepath.Join(stager.BuildDir(), Procfile), []byte(newCmd), 0755)
	if err != nil {
		sl.Log.Error("failed to update %s, error: %s", Procfile, err.Error())
		return err
	}

	return nil
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
	sb.WriteString("web: node ./node_modules/.bin/slnodejs run  --useinitialcolor true ")

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

func (h *SealightsHook) isSealightsBound() bool {
	type Service struct {
		Name string `json:"name"`
	}
	var vcapServices map[string][]Service
	err := json.Unmarshal([]byte(os.Getenv("VCAP_SERVICES")), &vcapServices)
	if err != nil {
		h.Log.Warning("Failed to parse VCAP_SERVICES")
		return false
	}

	for key := range vcapServices {
		if strings.Contains(key, "Sealights") {
			return true
		}
	}

	return false
}

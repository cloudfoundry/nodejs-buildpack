package supply

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

type Command interface {
	Execute(string, io.Writer, io.Writer, string, ...string) error
}

type Manifest interface {
	AllDependencyVersions(string) []string
	DefaultVersion(string) (libbuildpack.Dependency, error)
	InstallDependency(libbuildpack.Dependency, string) error
	InstallOnlyVersion(string, string) error
}

type Stager interface {
	BuildDir() string
	DepDir() string
	DepsIdx() string
	LinkDirectoryInDepDir(string, string) error
	WriteEnvFile(string, string) error
	WriteProfileD(string, string) error
}

type Supplier struct {
	Stager   Stager
	Manifest Manifest
	Log      *libbuildpack.Logger
	Command  Command
	Node     string
	Yarn     string
	NPM      string
}

type packageJSON struct {
	Engines engines `json:"engines"`
}

type engines struct {
	Node string `json:"node"`
	Yarn string `json:"yarn"`
	NPM  string `json:"npm"`
	Iojs string `json:"iojs"`
}

func Run(s *Supplier) error {
	s.Log.BeginStep("Installing binaries")
	if err := s.LoadPackageJSON(); err != nil {
		s.Log.Error("Unable to load package.json: %s", err.Error())
		return err
	}

	s.WarnNodeEngine()

	if err := s.InstallNode("/tmp/node"); err != nil {
		s.Log.Error("Unable to install node: %s", err.Error())
		return err
	}

	if err := s.InstallNPM(); err != nil {
		s.Log.Error("Unable to install npm: %s", err.Error())
		return err
	}

	if err := s.InstallYarn(); err != nil {
		s.Log.Error("Unable to install yarn: %s", err.Error())
		return err
	}

	if err := s.CreateDefaultEnv(); err != nil {
		s.Log.Error("Unable to setup default environment: %s", err.Error())
		return err
	}

	return nil
}

func (s *Supplier) LoadPackageJSON() error {
	var p packageJSON

	err := libbuildpack.NewJSON().Load(filepath.Join(s.Stager.BuildDir(), "package.json"), &p)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if p.Engines.Iojs != "" {
		return errors.New("io.js not supported by this buildpack")
	}

	if p.Engines.Node != "" {
		s.Log.Info("engines.node (package.json): %s", p.Engines.Node)
	} else {
		s.Log.Info("engines.node (package.json): unspecified")
	}

	if p.Engines.NPM != "" {
		s.Log.Info("engines.npm (package.json): %s", p.Engines.NPM)
	} else {
		s.Log.Info("engines.npm (package.json): unspecified (use default)")
	}

	s.Node = p.Engines.Node
	s.NPM = p.Engines.NPM
	s.Yarn = p.Engines.Yarn

	return nil
}

func (s *Supplier) WarnNodeEngine() {
	docsLink := "http://docs.cloudfoundry.org/buildpacks/node/node-tips.html"

	if s.Node == "" {
		s.Log.Warning("Node version not specified in package.json. See: %s", docsLink)
	}
	if s.Node == "*" {
		s.Log.Warning("Dangerous semver range (*) in engines.node. See: %s", docsLink)
	}
	if strings.HasPrefix(s.Node, ">") {
		s.Log.Warning("Dangerous semver range (>) in engines.node. See: %s", docsLink)
	}
	return
}

func (s *Supplier) InstallNode(tempDir string) error {
	var dep libbuildpack.Dependency

	nodeInstallDir := filepath.Join(s.Stager.DepDir(), "node")

	if s.Node != "" {
		versions := s.Manifest.AllDependencyVersions("node")
		ver, err := libbuildpack.FindMatchingVersion(s.Node, versions)
		if err != nil {
			return err
		}
		dep.Name = "node"
		dep.Version = ver
	} else {
		var err error

		dep, err = s.Manifest.DefaultVersion("node")
		if err != nil {
			return err
		}
	}

	if err := s.Manifest.InstallDependency(dep, tempDir); err != nil {
		return err
	}

	if err := os.Rename(filepath.Join(tempDir, fmt.Sprintf("node-v%s-linux-x64", dep.Version)), nodeInstallDir); err != nil {
		return err
	}

	if err := s.Stager.LinkDirectoryInDepDir(filepath.Join(nodeInstallDir, "bin"), "bin"); err != nil {
		return err
	}

	return os.Setenv("PATH", fmt.Sprintf("%s:%s", os.Getenv("PATH"), filepath.Join(s.Stager.DepDir(), "bin")))
}

func (s *Supplier) InstallNPM() error {
	buffer := new(bytes.Buffer)
	if err := s.Command.Execute(s.Stager.BuildDir(), buffer, buffer, "npm", "--version"); err != nil {
		return err
	}

	npmVersion := strings.TrimSpace(buffer.String())

	if s.NPM == "" {
		s.Log.Info("Using default npm version: %s", npmVersion)
		return nil
	}
	if s.NPM == npmVersion {
		s.Log.Info("npm %s already installed with node", npmVersion)
		return nil
	}

	s.Log.Info("Downloading and installing npm %s (replacing version %s)...", s.NPM, npmVersion)

	if err := s.Command.Execute(s.Stager.BuildDir(), ioutil.Discard, ioutil.Discard, "npm", "install", "--unsafe-perm", "--quiet", "-g", "npm@"+s.NPM); err != nil {
		s.Log.Error("We're unable to download the version of npm you've provided (%s).\nPlease remove the npm version specification in package.json", s.NPM)
		return err
	}
	return nil
}

func (s *Supplier) InstallYarn() error {
	if s.Yarn != "" {
		versions := s.Manifest.AllDependencyVersions("yarn")
		_, err := libbuildpack.FindMatchingVersion(s.Yarn, versions)
		if err != nil {
			return fmt.Errorf("package.json requested %s, buildpack only includes yarn version %s", s.Yarn, strings.Join(versions, ", "))
		}
	}

	yarnInstallDir := filepath.Join(s.Stager.DepDir(), "yarn")

	if err := s.Manifest.InstallOnlyVersion("yarn", yarnInstallDir); err != nil {
		return err
	}

	if err := s.Stager.LinkDirectoryInDepDir(filepath.Join(yarnInstallDir, "dist", "bin"), "bin"); err != nil {
		return err
	}

	buffer := new(bytes.Buffer)
	if err := s.Command.Execute(s.Stager.BuildDir(), buffer, buffer, "yarn", "--version"); err != nil {
		return err
	}

	yarnVersion := strings.TrimSpace(buffer.String())
	s.Log.Info("Installed yarn %s", yarnVersion)

	return nil
}

func (s *Supplier) CreateDefaultEnv() error {
	var environmentDefaults = map[string]string{
		"NODE_ENV":              "production",
		"NPM_CONFIG_PRODUCTION": "true",
		"NPM_CONFIG_LOGLEVEL":   "error",
		"NODE_MODULES_CACHE":    "true",
		"NODE_VERBOSE":          "false",
	}

	s.Log.BeginStep("Creating runtime environment")

	for envVar, envDefault := range environmentDefaults {
		if os.Getenv(envVar) == "" {
			if err := s.Stager.WriteEnvFile(envVar, envDefault); err != nil {
				return err
			}
		}
	}

	if err := s.Stager.WriteEnvFile("NODE_HOME", filepath.Join(s.Stager.DepDir(), "node")); err != nil {
		return err
	}

	scriptContents := `export NODE_HOME=%s
export NODE_ENV=${NODE_ENV:-production}
export MEMORY_AVAILABLE=$(echo $VCAP_APPLICATION | jq '.limits.mem')
export WEB_MEMORY=512
export WEB_CONCURRENCY=1
`

	return s.Stager.WriteProfileD("node.sh", fmt.Sprintf(scriptContents, filepath.Join("$DEPS_DIR", s.Stager.DepsIdx(), "node")))
}

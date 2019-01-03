package supply

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/Masterminds/semver"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/cloudfoundry/nodejs-buildpack/src/nodejs/package_json"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/checksum"
)

type Command interface {
	Execute(string, io.Writer, io.Writer, string, ...string) error
}

type Manifest interface {
	AllDependencyVersions(string) []string
	DefaultVersion(string) (libbuildpack.Dependency, error)
}

type Installer interface {
	InstallDependency(libbuildpack.Dependency, string) error
	InstallOnlyVersion(string, string) error
}

type NPM interface {
	Build(string, string) error
	Rebuild(string) error
}

type Yarn interface {
	Build(string, string) error
}

type Stager interface {
	BuildDir() string
	CacheDir() string
	DepDir() string
	DepsIdx() string
	LinkDirectoryInDepDir(string, string) error
	WriteEnvFile(string, string) error
	WriteProfileD(string, string) error
	SetStagingEnvironment() error
}

type Supplier struct {
	Stager                 Stager
	Manifest               Manifest
	Installer              Installer
	Log                    *libbuildpack.Logger
	Logfile                *os.File
	Command                Command
	NodeVersion            string
	PackageJSONNodeVersion string
	NvmrcNodeVersion       string
	YarnVersion            string
	NPMVersion             string
	PreBuild               string
	StartScript            string
	HasDevDependencies     bool
	PostBuild              string
	UseYarn                bool
	UsesYarnWorkspaces     bool
	IsVendored             bool
	Yarn                   Yarn
	NPM                    NPM
}

var LTS = map[string]int{
	"argon":   4,
	"boron":   6,
	"carbon":  8,
	"dubnium": 10,
}

func Run(s *Supplier) error {
	return checksum.Do(s.Stager.BuildDir(), s.Log.Debug, func() error {
		s.Log.BeginStep("Installing binaries")

		if err := s.LoadPackageJSON(); err != nil {
			s.Log.Error("Unable to load package.json: %s", err.Error())
			return err
		}

		if err := s.LoadNvmrc(); err != nil {
			s.Log.Error("Unable to load .nvmrc: %s", err.Error())
			return err
		}

		s.WarnNodeEngine()

		if err := s.ChooseNodeVersion(); err != nil {
			s.Log.Error("Unable to install node: %s", err.Error())
			return err
		}

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

		if err := s.Stager.SetStagingEnvironment(); err != nil {
			s.Log.Error("Unable to setup environment variables: %s", err.Error())
			os.Exit(11)
		}

		if err := s.ReadPackageJSON(); err != nil {
			s.Log.Error("Failed parsing package.json: %s", err.Error())
			return err
		}

		if err := s.TipVendorDependencies(); err != nil {
			s.Log.Error(err.Error())
			return err
		}

		if err := s.NoPackageLockTip(); err != nil {
			s.Log.Error(err.Error())
			return err
		}

		s.ListNodeConfig(os.Environ())

		if err := s.OverrideCacheFromApp(); err != nil {
			s.Log.Error("Unable to copy cache directories: %s", err.Error())
			return err
		}

		defer func() {
			s.Logfile.Sync()
			s.WarnUntrackedDependencies()
			s.WarnMissingDevDeps()
		}()

		if err := s.BuildDependencies(); err != nil {
			s.Log.Error("Unable to build dependencies: %s", err.Error())
			return err
		}

		if !s.UseYarn || !s.UsesYarnWorkspaces {
			if err := s.MoveDependencyArtifacts(); err != nil {
				s.Log.Error("Unable to move dependencies: %s", err.Error())
				return err
			}
		}

		s.ListDependencies()

		if err := s.Logfile.Sync(); err != nil {
			s.Log.Error(err.Error())
			return err
		}

		if err := s.WarnUnmetDependencies(); err != nil {
			s.Log.Error(err.Error())
			return err
		}

		return nil
	})
}

func (s *Supplier) WarnUnmetDependencies() error {
	if unmet, err := fileHasString(s.Logfile.Name(), "unmet dependency", "unmet peer dependency"); err != nil {
		return err
	} else if !unmet {
		return nil
	}

	pkgMan := "npm"
	if s.UseYarn {
		pkgMan = "yarn"
	}

	warning := "Unmet dependencies don't fail " + pkgMan + " install but may cause runtime issues\n"
	warning += "See: https://github.com/npm/npm/issues/7494"
	s.Log.Warning(warning)
	return nil
}

func (s *Supplier) ListDependencies() {
	if os.Getenv("NODE_VERBOSE") != "true" {
		return
	}

	if s.UseYarn {
		_ = s.Command.Execute(s.Stager.BuildDir(), s.Log.Output(), ioutil.Discard, "yarn", "list", "--depth=0")
	} else {
		_ = s.Command.Execute(s.Stager.BuildDir(), s.Log.Output(), ioutil.Discard, "npm", "ls", "--depth=0")
	}
}

func (s *Supplier) runPostbuild(tool string) error {
	if s.PostBuild == "" {
		return nil
	}

	return s.runScript("heroku-postbuild", tool)
}

func (s *Supplier) runScript(script, tool string) error {
	args := []string{"run", script}
	if tool == "npm" {
		args = append(args, "--if-present")
	}

	s.Log.Info("Running %s (%s)", script, tool)

	return s.Command.Execute(s.Stager.BuildDir(), os.Stdout, os.Stderr, tool, args...)

}

func (s *Supplier) runPrebuild(tool string) error {
	if s.PreBuild == "" {
		return nil
	}

	return s.runScript("heroku-prebuild", tool)
}

func (s *Supplier) BuildDependencies() error {
	var tool string
	if s.UseYarn {
		tool = "yarn"
	} else {
		tool = "npm"
	}

	s.Log.BeginStep("Building dependencies")

	if err := s.runPrebuild(tool); err != nil {
		return err
	}

	if s.UseYarn {
		if err := s.Yarn.Build(s.Stager.BuildDir(), s.Stager.CacheDir()); err != nil {
			return err
		}
	} else if s.IsVendored {
		s.Log.Info("Prebuild detected (node_modules already exists)")
		if err := s.NPM.Rebuild(s.Stager.BuildDir()); err != nil {
			return err
		}
	} else {
		if err := s.NPM.Build(s.Stager.BuildDir(), s.Stager.CacheDir()); err != nil {
			return err
		}
	}

	if err := s.runPostbuild(tool); err != nil {
		return err
	}

	return nil
}

func (s *Supplier) MoveDependencyArtifacts() error {
	if s.IsVendored {
		return nil
	}

	appNodeModules := filepath.Join(s.Stager.BuildDir(), "node_modules")

	_, err := os.Stat(appNodeModules)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	nodePath := filepath.Join(s.Stager.DepDir(), "node_modules")

	if err := os.Rename(appNodeModules, nodePath); err != nil {
		return err
	}

	if err := s.Stager.WriteEnvFile("NODE_PATH", nodePath); err != nil {
		return err
	}

	return os.Setenv("NODE_PATH", nodePath)
}

func (s *Supplier) ReadPackageJSON() error {
	var err error
	var p struct {
		Scripts struct {
			PreBuild    string `json:"heroku-prebuild"`
			PostBuild   string `json:"heroku-postbuild"`
			StartScript string `json:"start"`
		} `json:"scripts"`
		DevDependencies map[string]string `json:"devDependencies"`
		Workspaces      []string          `json:"workspaces"`
	}

	if s.UseYarn, err = libbuildpack.FileExists(filepath.Join(s.Stager.BuildDir(), "yarn.lock")); err != nil {
		return err
	}

	if s.IsVendored, err = libbuildpack.FileExists(filepath.Join(s.Stager.BuildDir(), "node_modules")); err != nil {
		return err
	}

	if err := libbuildpack.NewJSON().Load(filepath.Join(s.Stager.BuildDir(), "package.json"), &p); err != nil {
		if os.IsNotExist(err) {
			s.Log.Warning("No package.json found")
			return nil
		} else {
			return err
		}
	}

	s.UsesYarnWorkspaces = (len(p.Workspaces) > 0)
	s.HasDevDependencies = (len(p.DevDependencies) > 0)
	s.PreBuild = p.Scripts.PreBuild
	s.PostBuild = p.Scripts.PostBuild
	s.StartScript = p.Scripts.StartScript

	return nil
}

func (s *Supplier) NoPackageLockTip() error {
	var lockFiles []string
	if s.UseYarn {
		lockFiles = append(lockFiles, "yarn.lock")
	} else {
		lockFiles = append(lockFiles, "package-lock.json", "npm-shrinkwrap.json")
	}

	for _, lockFile := range lockFiles {
		if lockFileExists, err := libbuildpack.FileExists(filepath.Join(s.Stager.BuildDir(), lockFile)); err != nil {
			return err
		} else if lockFileExists {
			return nil
		}

		if s.IsVendored {
			s.Log.Protip("Warning: package-lock.json not found. The buildpack may reach out to the internet to download module updates, even if they are vendored.", "https://docs.cloudfoundry.org/buildpacks/node/index.html#offline_environments")
		}

		return nil
	}
	return nil
}

func (s *Supplier) TipVendorDependencies() error {
	subdirs, err := hasSubdirs(filepath.Join(s.Stager.BuildDir(), "node_modules"))
	if err != nil {
		return err
	}
	if !subdirs {
		s.Log.Protip("It is recommended to vendor the application's Node.js dependencies",
			"http://docs.cloudfoundry.org/buildpacks/node/index.html#vendoring")
	}

	return nil
}

func hasSubdirs(path string) (bool, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	for _, file := range files {
		if file.IsDir() {
			return true, nil
		}
	}

	return false, nil
}

func (s *Supplier) ListNodeConfig(environment []string) {
	npmConfigProductionTrue := false
	nodeEnv := "production"

	for _, env := range environment {
		if strings.HasPrefix(env, "NPM_CONFIG_") || strings.HasPrefix(env, "YARN_") || strings.HasPrefix(env, "NODE_") {
			s.Log.Info(env)
		}

		if env == "NPM_CONFIG_PRODUCTION=true" {
			npmConfigProductionTrue = true
		}

		if strings.HasPrefix(env, "NODE_ENV=") {
			nodeEnv = env[9:]
		}
	}

	if npmConfigProductionTrue && nodeEnv != "production" {
		s.Log.Info("npm scripts will see NODE_ENV=production (not '%s')\nhttps://docs.npmjs.com/misc/config#production", nodeEnv)
	}
}

func (s *Supplier) WarnUntrackedDependencies() error {
	for _, command := range []string{"grunt", "bower", "gulp"} {
		if notFound, err := fileHasString(s.Logfile.Name(), command+": not found", command+": command not found"); err != nil {
			return err
		} else if notFound {
			s.Log.Warning(strings.Title(command) + " may not be tracked in package.json")
		}
	}

	return nil
}

func (s *Supplier) WarnMissingDevDeps() error {
	if noModule, err := fileHasString(s.Logfile.Name(), "cannot find module"); err != nil {
		return err
	} else if !noModule {
		return nil
	}

	warning := "A module may be missing from 'dependencies' in package.json"

	if os.Getenv("NPM_CONFIG_PRODUCTION") == "true" && s.HasDevDependencies {
		warning += "\nThis module may be specified in 'devDependencies' instead of 'dependencies'\n"
		warning += "See: https://devcenter.heroku.com/articles/nodejs-support#devdependencies"
	}

	s.Log.Warning(warning)
	return nil
}

func fileHasString(file string, patterns ...string) (bool, error) {
	f, err := os.Open(file)
	if err != nil {
		return false, err
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	line, err := reader.ReadString('\n')
	for err == nil {
		src := strings.ToLower(line)
		for _, pat := range patterns {
			if strings.Contains(src, pat) {
				return true, nil
			}
		}
		line, err = reader.ReadString('\n')
	}
	if err != io.EOF {
		return false, err
	}

	return false, nil
}

func (s *Supplier) LoadPackageJSON() error {
	p, err := package_json.LoadPackageJSON(filepath.Join(s.Stager.BuildDir(), "package.json"), s.Log)
	if err != nil {
		return err
	}

	s.PackageJSONNodeVersion = p.Engines.Node
	s.NPMVersion = p.Engines.NPM
	s.YarnVersion = p.Engines.Yarn

	return nil
}

func formatNvmrcContent(version string) string {
	if version == "node" {
		return "*"
	} else if strings.HasPrefix(version, "lts") {
		ltsName := strings.Split(version, "/")[1]
		if ltsName == "*" {
			maxVersion := 0
			for _, versionValue := range LTS {
				if maxVersion < versionValue {
					maxVersion = versionValue
				}
			}
			return strconv.Itoa(maxVersion) + ".*.*"
		} else {
			versionNumber := LTS[ltsName]
			return strconv.Itoa(versionNumber) + ".*.*"
		}
	} else {
		matcher := regexp.MustCompile(semver.SemVerRegex)

		groups := matcher.FindStringSubmatch(version)
		for index := 0; index < len(groups); index++ {
			if groups[index] == "" {
				groups = append(groups[:index], groups[index+1:]...)
				index--
			}
		}

		return version + strings.Repeat(".*", 4-len(groups))
	}
}

func (s *Supplier) LoadNvmrc() error {
	if nvmrcExists, err := libbuildpack.FileExists(filepath.Join(s.Stager.BuildDir(), ".nvmrc")); err != nil {
		return err
	} else if !nvmrcExists {
		return nil
	}

	nvmrcContents, err := ioutil.ReadFile(filepath.Join(s.Stager.BuildDir(), ".nvmrc"))
	if err != nil {
		return err
	}

	nvmrcVersion, err := validateNvmrc(string(nvmrcContents))
	if err != nil {
		return err
	}

	s.NvmrcNodeVersion = formatNvmrcContent(nvmrcVersion)

	return nil
}

func (s *Supplier) ChooseNodeVersion() error {
	var (
		selectedVersion string
		err             error
	)

	versions := s.Manifest.AllDependencyVersions("node")

	if s.PackageJSONNodeVersion != "" {
		if selectedVersion, err = libbuildpack.FindMatchingVersion(s.PackageJSONNodeVersion, versions); err != nil {
			return err
		}
	} else if s.NvmrcNodeVersion != "" {
		if selectedVersion, err = libbuildpack.FindMatchingVersion(s.NvmrcNodeVersion, versions); err != nil {
			return err
		}
	} else {
		if dep, err := s.Manifest.DefaultVersion("node"); err != nil {
			return err
		} else {
			selectedVersion = dep.Version
		}
	}

	s.NodeVersion = selectedVersion

	return nil
}

func (s *Supplier) WarnNodeEngine() {
	docsLink := "http://docs.cloudfoundry.org/buildpacks/node/node-tips.html"

	if s.NvmrcNodeVersion != "" && s.PackageJSONNodeVersion == "" {
		s.Log.Warning("Using the node version specified in your .nvmrc See: %s", docsLink)
	}
	if s.PackageJSONNodeVersion != "" && s.NvmrcNodeVersion != "" {
		s.Log.Warning("Node version in .nvmrc ignored in favor of 'engines' field in package.json")
	}
	if s.PackageJSONNodeVersion == "" && s.NvmrcNodeVersion == "" {
		s.Log.Warning("Node version not specified in package.json or .nvmrc. See: %s", docsLink)
	}
	if s.PackageJSONNodeVersion == "*" {
		s.Log.Warning("Dangerous semver range (*) in engines.node. See: %s", docsLink)
	}
	if s.NvmrcNodeVersion == "node" {
		s.Log.Warning(".nvmrc specified latest node version, this will be selected from versions available in manifest.yml")
	}

	if strings.HasPrefix(s.NvmrcNodeVersion, "lts") {
		s.Log.Warning(".nvmrc specified an lts version, this will be selected from versions available in manifest.yml")
	}

	if strings.HasPrefix(s.PackageJSONNodeVersion, ">") {
		s.Log.Warning("Dangerous semver range (>) in engines.node. See: %s", docsLink)
	}
}

func (s *Supplier) InstallNode(tempDir string) error {
	var dep libbuildpack.Dependency

	nodeInstallDir := filepath.Join(s.Stager.DepDir(), "node")

	dep.Name = "node"
	dep.Version = s.NodeVersion

	if err := s.Installer.InstallDependency(dep, tempDir); err != nil {
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

	if s.NPMVersion == "" {
		s.Log.Info("Using default npm version: %s", npmVersion)
		return nil
	}

	_, err := libbuildpack.FindMatchingVersion(s.NPMVersion, []string{npmVersion})
	if err == nil {
		s.Log.Info("npm %s already installed with node", npmVersion)
		return nil
	}

	s.Log.Info("Downloading and installing npm %s (replacing version %s)...", s.NPMVersion, npmVersion)

	if err := s.Command.Execute(s.Stager.BuildDir(), ioutil.Discard, ioutil.Discard, "npm", "install", "--unsafe-perm", "--quiet", "-g", "npm@"+s.NPMVersion); err != nil {
		s.Log.Error("We're unable to download the version of npm you've provided (%s).\nPlease remove the npm version specification in package.json", s.NPMVersion)
		return err
	}
	return nil
}

func (s *Supplier) InstallYarn() error {
	if s.YarnVersion != "" {
		versions := s.Manifest.AllDependencyVersions("yarn")
		_, err := libbuildpack.FindMatchingVersion(s.YarnVersion, versions)
		if err != nil {
			return fmt.Errorf("package.json requested %s, buildpack only includes yarn version %s", s.YarnVersion, strings.Join(versions, ", "))
		}
	}

	yarnInstallDir := filepath.Join(s.Stager.DepDir(), "yarn")

	if err := s.Installer.InstallOnlyVersion("yarn", yarnInstallDir); err != nil {
		return err
	}

	if paths, err := filepath.Glob(filepath.Join(yarnInstallDir, "yarn-v*")); err != nil {
		return fmt.Errorf("Unable to find yarn distribution dir: %v", err)
	} else if len(paths) != 1 {
		return fmt.Errorf("Unable to find yarn distribution dir")
	} else {
		yarnInstallDir = paths[0]
	}

	if err := s.Stager.LinkDirectoryInDepDir(filepath.Join(yarnInstallDir, "bin"), "bin"); err != nil {
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
		"WEB_MEMORY":            "512",
		"WEB_CONCURRENCY":       "1",
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

	scriptContents := `export NODE_HOME=%[1]s
export NODE_ENV=${NODE_ENV:-production}
export MEMORY_AVAILABLE=$(echo $VCAP_APPLICATION | jq '.limits.mem')
export WEB_MEMORY=${WEB_MEMORY:-512}
export WEB_CONCURRENCY=${WEB_CONCURRENCY:-1}
if [ ! -d "$HOME/node_modules" ]; then
	export NODE_PATH=${NODE_PATH:-"%[2]s"}
	ln -s "%[2]s" "$HOME/node_modules"
else
	export NODE_PATH=${NODE_PATH:-"$HOME/node_modules"}
fi
export PATH=$PATH:"$HOME/bin":$NODE_PATH/.bin
`
	return s.Stager.WriteProfileD("node.sh",
		fmt.Sprintf(scriptContents,
			filepath.Join("$DEPS_DIR", s.Stager.DepsIdx(), "node"),
			filepath.Join("$DEPS_DIR", s.Stager.DepsIdx(), "node_modules")))
}

func copyAll(srcDir, destDir string, files []string) error {
	for _, filename := range files {
		fi, err := os.Stat(filepath.Join(srcDir, filename))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if fi.IsDir() {
			if err := os.RemoveAll(filepath.Join(destDir, filename)); err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Join(destDir, filename), 0755); err != nil {
				return err
			}
			if err := libbuildpack.CopyDirectory(filepath.Join(srcDir, filename), filepath.Join(destDir, filename)); err != nil && !os.IsNotExist(err) {
				return err
			}
		} else {
			if err := libbuildpack.CopyFile(filepath.Join(srcDir, filename), filepath.Join(destDir, filename)); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}
	return nil
}

func (s *Supplier) OverrideCacheFromApp() error {
	deprecatedCacheDirs := []string{"bower_components"}
	for _, name := range deprecatedCacheDirs {
		os.RemoveAll(filepath.Join(s.Stager.CacheDir(), name))
	}

	pkgMgrCacheDirs := []string{".cache/yarn", ".npm"}
	if err := copyAll(s.Stager.BuildDir(), s.Stager.CacheDir(), pkgMgrCacheDirs); err != nil {
		return err
	}

	return nil
}

func validateNvmrc(content string) (string, error) {
	content = strings.TrimSpace(strings.ToLower(content))

	if content == "lts/*" || content == "node" {
		return content, nil
	}

	for key, _ := range LTS {
		if content == strings.ToLower("lts/"+key) {
			return content, nil
		}
	}

	if len(content) > 0 && content[0] == 'v' {
		content = content[1:]
	}

	if _, err := semver.NewVersion(content); err != nil {
		return "", fmt.Errorf("invalid version %s specified in .nvmrc", err)
	}

	return content, nil
}

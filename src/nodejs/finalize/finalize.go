package finalize

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

type Cache interface {
	Initialize() error
	Restore() error
	Save() error
}

type Command interface {
	Execute(string, io.Writer, io.Writer, string, ...string) error
}

type Manifest interface {
	RootDir() string
}

type NPM interface {
	Build() error
	Rebuild() error
}

type Stager interface {
	BuildDir() string
	DepDir() string
}

type Yarn interface {
	Build() error
}

type Finalizer struct {
	Stager             Stager
	Command            Command
	Cache              Cache
	Log                *libbuildpack.Logger
	Logfile            *os.File
	PreBuild           string
	PostBuild          string
	NPM                NPM
	NPMRebuild         bool
	Yarn               Yarn
	UseYarn            bool
	Manifest           Manifest
	StartScript        string
	HasDevDependencies bool
}

func Run(f *Finalizer) error {
	if err := f.ReadPackageJSON(); err != nil {
		f.Log.Error("Failed parsing package.json: %s", err.Error())
		return err
	}

	if err := f.TipVendorDependencies(); err != nil {
		f.Log.Error(err.Error())
		return err
	}

	f.ListNodeConfig(os.Environ())

	if err := f.Cache.Initialize(); err != nil {
		f.Log.Error("Unable to initialize cache: %s", err.Error())
		return err
	}

	if err := f.Cache.Restore(); err != nil {
		f.Log.Error("Unable to restore cache: %s", err.Error())
		return err
	}

	defer func() {
		f.Logfile.Sync()
		f.WarnUntrackedDependencies()
		f.WarnMissingDevDeps()
	}()

	if err := f.BuildDependencies(); err != nil {
		f.Log.Error("Unable to build dependencies: %s", err.Error())
		return err
	}

	if err := f.Cache.Save(); err != nil {
		f.Log.Error("Unable to save cache: %s", err.Error())
		return err
	}

	if err := f.CopyProfileScripts(); err != nil {
		f.Log.Error("Unable to copy profile.d scripts: %s", err.Error())
		return err
	}

	f.ListDependencies()

	if err := f.WarnNoStart(); err != nil {
		f.Log.Error(err.Error())
		return err
	}

	if err := f.Logfile.Sync(); err != nil {
		f.Log.Error(err.Error())
		return err
	}

	if err := f.WarnUnmetDependencies(); err != nil {
		f.Log.Error(err.Error())
		return err
	}

	return nil
}

func (f *Finalizer) ReadPackageJSON() error {
	var err error
	var p struct {
		Scripts struct {
			PreBuild    string `json:"heroku-prebuild"`
			PostBuild   string `json:"heroku-postbuild"`
			StartScript string `json:"start"`
		} `json:"scripts"`
		DevDependencies map[string]string `json:"devDependencies"`
	}

	if f.UseYarn, err = libbuildpack.FileExists(filepath.Join(f.Stager.BuildDir(), "yarn.lock")); err != nil {
		return err
	}

	if f.NPMRebuild, err = libbuildpack.FileExists(filepath.Join(f.Stager.BuildDir(), "node_modules")); err != nil {
		return err
	}

	if err := libbuildpack.NewJSON().Load(filepath.Join(f.Stager.BuildDir(), "package.json"), &p); err != nil {
		if os.IsNotExist(err) {
			f.Log.Warning("No package.json found")
			return nil
		} else {
			return err
		}
	}

	f.HasDevDependencies = (len(p.DevDependencies) > 0)
	f.PreBuild = p.Scripts.PreBuild
	f.PostBuild = p.Scripts.PostBuild
	f.StartScript = p.Scripts.StartScript

	return nil
}

func (f *Finalizer) TipVendorDependencies() error {
	subdirs, err := hasSubdirs(filepath.Join(f.Stager.BuildDir(), "node_modules"))
	if err != nil {
		return err
	}
	if !subdirs {
		f.Log.Protip("It is recommended to vendor the application's Node.js dependencies",
			"http://docs.cloudfoundry.org/buildpacks/node/index.html#vendoring")
	}

	return nil
}

func (f *Finalizer) ListNodeConfig(environment []string) {
	npmConfigProductionTrue := false
	nodeEnv := "production"

	for _, env := range environment {
		if strings.HasPrefix(env, "NPM_CONFIG_") || strings.HasPrefix(env, "YARN_") || strings.HasPrefix(env, "NODE_") {
			f.Log.Info(env)
		}

		if env == "NPM_CONFIG_PRODUCTION=true" {
			npmConfigProductionTrue = true
		}

		if strings.HasPrefix(env, "NODE_ENV=") {
			nodeEnv = env[9:]
		}
	}

	if npmConfigProductionTrue && nodeEnv != "production" {
		f.Log.Info("npm scripts will see NODE_ENV=production (not '%s')\nhttps://docs.npmjs.com/misc/config#production", nodeEnv)
	}
}

func (f *Finalizer) BuildDependencies() error {
	var tool string
	if f.UseYarn {
		tool = "yarn"
	} else {
		tool = "npm"
	}

	f.Log.BeginStep("Building dependencies")

	if err := f.runPrebuild(tool); err != nil {
		return err
	}

	if f.UseYarn {
		if err := f.Yarn.Build(); err != nil {
			return err
		}
	} else {
		if f.NPMRebuild {
			f.Log.Info("Prebuild detected (node_modules already exists)")
			if err := f.NPM.Rebuild(); err != nil {
				return err
			}
		} else {
			if err := f.NPM.Build(); err != nil {
				return err
			}
		}
	}

	if err := f.runPostbuild(tool); err != nil {
		return err
	}

	return nil
}

func (f *Finalizer) CopyProfileScripts() error {
	profiledDir := filepath.Join(f.Stager.DepDir(), "profile.d")
	if err := os.MkdirAll(profiledDir, 0755); err != nil {
		return err
	}
	return libbuildpack.CopyDirectory(filepath.Join(f.Manifest.RootDir(), "profile"), profiledDir)
}

func (f *Finalizer) ListDependencies() {
	if os.Getenv("NODE_VERBOSE") != "true" {
		return
	}

	if f.UseYarn {
		_ = f.Command.Execute(f.Stager.BuildDir(), f.Log.Output(), ioutil.Discard, "yarn", "list", "--depth=0")
	} else {
		_ = f.Command.Execute(f.Stager.BuildDir(), f.Log.Output(), ioutil.Discard, "npm", "ls", "--depth=0")
	}
}

func (f *Finalizer) WarnNoStart() error {
	procfileExists, err := libbuildpack.FileExists(filepath.Join(f.Stager.BuildDir(), "Procfile"))
	if err != nil {
		return err
	}
	serverJsExists, err := libbuildpack.FileExists(filepath.Join(f.Stager.BuildDir(), "server.js"))
	if err != nil {
		return err
	}

	if !procfileExists && !serverJsExists && f.StartScript == "" {
		warning := "This app may not specify any way to start a node process\n"
		warning += "See: https://docs.cloudfoundry.org/buildpacks/node/node-tips.html#start"
		f.Log.Warning(warning)
	}

	return nil
}

func (f *Finalizer) WarnUnmetDependencies() error {
	if unmet, err := fileHasString(f.Logfile.Name(), "unmet dependency", "unmet peer dependency"); err != nil {
		return err
	} else if !unmet {
		return nil
	}

	pkgMan := "npm"
	if f.UseYarn {
		pkgMan = "yarn"
	}

	warning := "Unmet dependencies don't fail " + pkgMan + " install but may cause runtime issues\n"
	warning += "See: https://github.com/npm/npm/issues/7494"
	f.Log.Warning(warning)
	return nil
}

func (f *Finalizer) WarnUntrackedDependencies() error {
	for _, command := range []string{"grunt", "bower", "gulp"} {
		if notFound, err := fileHasString(f.Logfile.Name(), command+": not found", command+": command not found"); err != nil {
			return err
		} else if notFound {
			f.Log.Warning(strings.Title(command) + " may not be tracked in package.json")
		}
	}

	return nil
}

func (f *Finalizer) WarnMissingDevDeps() error {
	if noModule, err := fileHasString(f.Logfile.Name(), "cannot find module"); err != nil {
		return err
	} else if !noModule {
		return nil
	}

	warning := "A module may be missing from 'dependencies' in package.json"

	if os.Getenv("NPM_CONFIG_PRODUCTION") == "true" && f.HasDevDependencies {
		warning += "\nThis module may be specified in 'devDependencies' instead of 'dependencies'\n"
		warning += "See: https://devcenter.heroku.com/articles/nodejs-support#devdependencies"
	}

	f.Log.Warning(warning)
	return nil
}

func (f *Finalizer) runPrebuild(tool string) error {
	if f.PreBuild == "" {
		return nil
	}

	return f.runScript(f.PreBuild, tool)
}

func (f *Finalizer) runPostbuild(tool string) error {
	if f.PostBuild == "" {
		return nil
	}

	return f.runScript(f.PostBuild, tool)
}

func (f *Finalizer) runScript(script, tool string) error {
	args := []string{"run", script}
	if tool == "npm" {
		args = append(args, "--if-present")
	}

	f.Log.Info("Running %s (%s)", script, tool)

	return f.Command.Execute(f.Stager.BuildDir(), os.Stdout, os.Stderr, tool, args...)

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

func (f *Finalizer) findVersion(binary string) (string, error) {
	buffer := new(bytes.Buffer)
	if err := f.Command.Execute("", buffer, ioutil.Discard, binary, "--version"); err != nil {
		return "", err
	}
	return strings.TrimSpace(buffer.String()), nil
}

func fileHasString(file string, patterns ...string) (bool, error) {
	f, err := os.Open(file)
	if err != nil {
		return false, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		for _, pat := range patterns {
			src := strings.ToLower(scanner.Text())

			if strings.Contains(src, pat) {
				return true, nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}
	return false, nil
}

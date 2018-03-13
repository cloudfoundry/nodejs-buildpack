package finalize

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

type Manifest interface {
	RootDir() string
}

type Stager interface {
	BuildDir() string
	DepDir() string
	DepsIdx() string
}

type Finalizer struct {
	Stager      Stager
	Log         *libbuildpack.Logger
	Logfile     *os.File
	Manifest    Manifest
	StartScript string
}

func Run(f *Finalizer) error {
	if err := f.MoveNodeModulesToHome(); err != nil {
		f.Log.Error("Failed to move node_modules directory back to app dir: %s", err.Error())
		return err
	}

	if err := f.ReadPackageJSON(); err != nil {
		f.Log.Error("Failed parsing package.json: %s", err.Error())
		return err
	}

	if err := f.CopyProfileScripts(); err != nil {
		f.Log.Error("Unable to copy profile.d scripts: %s", err.Error())
		return err
	}

	if err := f.WarnNoStart(); err != nil {
		f.Log.Error(err.Error())
		return err
	}

	if err := f.Logfile.Sync(); err != nil {
		f.Log.Error(err.Error())
		return err
	}

	return nil
}

func (f *Finalizer) MoveNodeModulesToHome() error {
	if err := os.Unsetenv("NODE_PATH"); err != nil {
		return err
	}
	pkgDir := filepath.Join(f.Stager.DepDir(), "packages")
	if exist, err := libbuildpack.FileExists(filepath.Join(pkgDir, "node_modules")); err != nil {
		return err
	} else if exist {
		os.RemoveAll(filepath.Join(f.Stager.BuildDir(), "node_modules"))
		if err := os.Rename(filepath.Join(pkgDir, "node_modules"), filepath.Join(f.Stager.BuildDir(), "node_modules")); err != nil {
			return err
		}
	}
	return os.RemoveAll(pkgDir)
}

func (f *Finalizer) ReadPackageJSON() error {
	var p struct {
		Scripts struct {
			StartScript string `json:"start"`
		} `json:"scripts"`
	}

	if err := libbuildpack.NewJSON().Load(filepath.Join(f.Stager.BuildDir(), "package.json"), &p); err != nil {
		if os.IsNotExist(err) {
			f.Log.Warning("No package.json found")
			return nil
		} else {
			return err
		}
	}

	f.StartScript = p.Scripts.StartScript

	return nil
}

func (f *Finalizer) CopyProfileScripts() error {
	profiledDir := filepath.Join(f.Stager.DepDir(), "profile.d")
	if err := os.MkdirAll(profiledDir, 0755); err != nil {
		return err
	}

	scriptsDir := filepath.Join(f.Stager.DepDir(), "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		return err
	}

	path := filepath.Join(f.Manifest.RootDir(), "profile")
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}

	for _, fi := range files {
		if strings.HasSuffix(fi.Name(), ".rb") {
			if err := libbuildpack.CopyFile(filepath.Join(path, fi.Name()), filepath.Join(scriptsDir, fi.Name())); err != nil {
				return err
			}
			if err := ioutil.WriteFile(filepath.Join(profiledDir, fi.Name()+".sh"), []byte("eval $(ruby $DEPS_DIR/"+f.Stager.DepsIdx()+"/scripts/"+fi.Name()+")\n"), 0755); err != nil {
				return err
			}
		} else {
			if err := libbuildpack.CopyFile(filepath.Join(path, fi.Name()), filepath.Join(profiledDir, fi.Name())); err != nil {
				return err
			}
		}
	}
	return nil
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

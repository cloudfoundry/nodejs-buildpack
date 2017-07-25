package finalize

import (
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
)

type Manifest interface {
	RootDir() string
}

type Stager interface {
	BuildDir() string
	DepDir() string
}

type Finalizer struct {
	Stager      Stager
	Log         *libbuildpack.Logger
	Logfile     *os.File
	Manifest    Manifest
	StartScript string
}

func Run(f *Finalizer) error {
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
	return libbuildpack.CopyDirectory(filepath.Join(f.Manifest.RootDir(), "profile"), profiledDir)
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

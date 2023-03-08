package main

import (
	"io"
	"os"
	"time"

	_ "github.com/cloudfoundry/nodejs-buildpack/src/nodejs/hooks"
	"github.com/cloudfoundry/nodejs-buildpack/src/nodejs/npm"
	"github.com/cloudfoundry/nodejs-buildpack/src/nodejs/supply"
	"github.com/cloudfoundry/nodejs-buildpack/src/nodejs/yarn"

	"github.com/cloudfoundry/libbuildpack"
)

func main() {
	logfile, err := os.CreateTemp("", "cloudfoundry.nodejs-buildpack.supply")
	if err != nil {
		logger := libbuildpack.NewLogger(os.Stdout)
		logger.Error("Unable to create log file: %s", err.Error())
		os.Exit(8)
	}
	defer logfile.Close()

	stdout := io.MultiWriter(os.Stdout, logfile)
	logger := libbuildpack.NewLogger(stdout)

	buildpackDir, err := libbuildpack.GetBuildpackDir()
	if err != nil {
		logger.Error("Unable to determine buildpack directory: %s", err.Error())
		os.Exit(9)
	}

	manifest, err := libbuildpack.NewManifest(buildpackDir, logger, time.Now())
	if err != nil {
		logger.Error("Unable to load buildpack manifest: %s", err.Error())
		os.Exit(10)
	}
	installer := libbuildpack.NewInstaller(manifest)

	stager := libbuildpack.NewStager(os.Args[1:], logger, manifest)
	if err := stager.CheckBuildpackValid(); err != nil {
		os.Exit(11)
	}

	if err = installer.SetAppCacheDir(stager.CacheDir()); err != nil {
		logger.Error("Unable to setup appcache: %s", err)
		os.Exit(18)
	}
	if err = manifest.ApplyOverride(stager.DepsDir()); err != nil {
		logger.Error("Unable to apply override.yml files: %s", err)
		os.Exit(17)
	}

	err = libbuildpack.RunBeforeCompile(stager)
	if err != nil {
		logger.Error("Before Compile: %s", err.Error())
		os.Exit(12)
	}

	err = stager.SetStagingEnvironment()
	if err != nil {
		logger.Error("Unable to setup environment variables: %s", err.Error())
		os.Exit(13)
	}

	s := supply.Supplier{
		Logfile: logfile,
		Stager:  stager,
		Yarn: &yarn.Yarn{
			Command: &libbuildpack.Command{},
			Log:     logger,
		},
		NPM: &npm.NPM{
			Command: &libbuildpack.Command{},
			Log:     logger,
		},
		Manifest:  manifest,
		Installer: installer,
		Log:       logger,
		Command:   &libbuildpack.Command{},
	}

	err = supply.Run(&s)
	if err != nil {
		os.Exit(14)
	}

	if err := stager.WriteConfigYml(nil); err != nil {
		logger.Error("Error writing config.yml: %s", err.Error())
		os.Exit(15)
	}
	if err = installer.CleanupAppCache(); err != nil {
		logger.Error("Unable to clean up app cache: %s", err)
		os.Exit(19)
	}
}

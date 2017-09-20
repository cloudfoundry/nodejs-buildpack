package main

import (
	"io"
	"io/ioutil"
	"nodejs/cache"
	_ "nodejs/hooks"
	"nodejs/npm"
	"nodejs/supply"
	"nodejs/yarn"
	"os"
	"time"

	"github.com/cloudfoundry/libbuildpack"
)

func main() {
	logfile, err := ioutil.TempFile("", "cloudfoundry.nodejs-buildpack.finalize")
	defer logfile.Close()
	if err != nil {
		logger := libbuildpack.NewLogger(os.Stdout)
		logger.Error("Unable to create log file: %s", err.Error())
		os.Exit(8)
	}

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

	stager := libbuildpack.NewStager(os.Args[1:], logger, manifest)
	if err := stager.CheckBuildpackValid(); err != nil {
		os.Exit(11)
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
			BuildDir: stager.BuildDir(),
			Command:  &libbuildpack.Command{},
			Log:      logger,
		},
		NPM: &npm.NPM{
			BuildDir: stager.BuildDir(),
			Command:  &libbuildpack.Command{},
			Log:      logger,
		},
		Manifest: manifest,
		Log:      logger,
		Command:  &libbuildpack.Command{},
		Cache: &cache.Cache{
			Stager:  stager,
			Command: &libbuildpack.Command{},
			Log:     logger,
		},
	}

	err = supply.Run(&s)
	if err != nil {
		os.Exit(14)
	}

	if err := stager.WriteConfigYml(nil); err != nil {
		logger.Error("Error writing config.yml: %s", err.Error())
		os.Exit(15)
	}
}

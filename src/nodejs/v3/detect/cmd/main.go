package main

import (
	"fmt"
	libbuildpackV3 "github.com/buildpack/libbuildpack"
	"nodejs/v3/detect"
	"os"
)

func main() {
	detector, err := libbuildpackV3.DefaultDetect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create default detector: %s", err)
		os.Exit(100)
	}

	if err := detect.UpdateBuildPlan(&detector); err != nil {
		detector.Logger.Debug("failed nodejs detection: %s", err)
		detector.Fail()
	}

	detector.Pass(detector.BuildPlan)
}

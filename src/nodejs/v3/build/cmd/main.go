package main

import (
	"fmt"
	libbuildpackV3 "github.com/buildpack/libbuildpack"
	"nodejs/v3/build"
	"os"
)

func main() {
	builder, err := libbuildpackV3.DefaultBuild()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create default builder: %s", err)
		os.Exit(100)
	}

	if err := builder.Launch.WriteMetadata(build.CreateLaunchMetadata()); err != nil {
		builder.Logger.Info("failed nodejs build: %s", err)
		builder.Failure(100)
	}
}

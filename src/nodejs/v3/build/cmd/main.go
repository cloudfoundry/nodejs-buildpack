package main

import (
	"fmt"
	"github.com/cloudfoundry/libjavabuildpack"
	"nodejs/v3/build"
	"os"
)

func main() {
	builder, err := libjavabuildpack.DefaultBuild()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create default builder: %s", err)
		os.Exit(100)
	}

	node, ok, err := build.NewNode(builder)
	if err != nil {
		builder.Logger.Info(err.Error())
		builder.Failure(102)
		return
	}

	if ok {
		if err := node.Contribute(); err != nil {
			builder.Logger.Info(err.Error())
			builder.Failure(103)
			return
		}
	}

	if err := builder.Launch.WriteMetadata(build.CreateLaunchMetadata()); err != nil {
		builder.Logger.Info("failed node build: %s", err)
		builder.Failure(100)
	}

	builder.Success()
}

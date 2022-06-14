package cloudfoundry

import (
	"bytes"
	"fmt"

	"github.com/paketo-buildpacks/packit/v2/pexec"
)

type Buildpack struct {
	Name string
	URI  string
}

type InitializePhase interface {
	Run([]Buildpack) error
}

type Initialize struct {
	cli Executable
}

func NewInitialize(cli Executable) Initialize {
	return Initialize{cli: cli}
}

func (i Initialize) Run(buildpacks []Buildpack) error {
	logs := bytes.NewBuffer(nil)

	for _, buildpack := range buildpacks {
		err := i.cli.Execute(pexec.Execution{
			Args:   []string{"update-buildpack", buildpack.Name, "-p", buildpack.URI},
			Stdout: logs,
			Stderr: logs,
		})
		if err == nil {
			continue
		}

		err = i.cli.Execute(pexec.Execution{
			Args:   []string{"create-buildpack", buildpack.Name, buildpack.URI, "1000"},
			Stdout: logs,
			Stderr: logs,
		})
		if err != nil {
			return fmt.Errorf("failed to create buildpack: %w\n\nOutput:\n%s", err, logs)
		}
	}

	return nil
}

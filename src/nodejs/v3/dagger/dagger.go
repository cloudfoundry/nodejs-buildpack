package dagger

import (
	"fmt"
	"github.com/BurntSushi/toml"
	libbuildpackV3 "github.com/buildpack/libbuildpack"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

type Dagger struct {
	rootDir, workspaceDir, buildpackDir string
}

func NewDagger(rootDir string) (*Dagger, error) {
	buildpackDir, err := ioutil.TempDir("/tmp", "buildpack")
	if err != nil {
		return nil, err
	}

	workspaceDir, err := ioutil.TempDir("/tmp", "workspace")
	if err != nil {
		return nil, err
	}

	if err := os.Chmod(workspaceDir, os.ModePerm); err != nil {
		return nil, err
	}

	return &Dagger{
		rootDir:      rootDir,
		workspaceDir: workspaceDir,
		buildpackDir: buildpackDir,
	}, nil
}

func (d *Dagger) Destroy() {
	os.RemoveAll(d.workspaceDir)
	d.workspaceDir = ""

	os.RemoveAll(d.buildpackDir)
	d.buildpackDir = ""
}

func (d *Dagger) BundleBuildpack() error {
	if err := copyFile(filepath.Join(d.rootDir, "buildpack.toml"), filepath.Join(d.buildpackDir, "buildpack.toml")); err != nil {
		return err
	}

	if err := os.Mkdir(filepath.Join(d.buildpackDir, "bin"), os.ModePerm); err != nil {
		return err
	}

	for _, b := range []string{"detect", "build"} {
		cmd := exec.Command(
			"go",
			"build",
			"-o",
			filepath.Join(d.buildpackDir, "bin", b),
			filepath.Join("nodejs", "v3", b, "cmd"),
		)
		cmd.Env = append(os.Environ(), "GOPATH="+d.rootDir, "GOOS=linux")
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

type DetectResult struct {
	Group struct {
		Buildpacks []struct {
			Id      string `toml:"id"`
			Version string `toml:"version"`
		} `toml:"buildpacks"`
	}
	BuildPlan libbuildpackV3.BuildPlan
}

func (d *Dagger) Detect(appDir string) (*DetectResult, error) {
	cmd := exec.Command(
		"docker",
		"run",
		"--rm",
		"-v",
		fmt.Sprintf("%s:/workspace", d.workspaceDir),
		"-v",
		fmt.Sprintf("%s:/workspace/app", appDir),
		"-v",
		fmt.Sprintf("%s:/buildpacks/org.cloudfoundry.buildpacks.nodejs/latest", d.buildpackDir),
		"-v",
		fmt.Sprintf("%s:/buildpacks/org.cloudfoundry.buildpacks.nodejs/1.6.32", d.buildpackDir),
		"-v",
		fmt.Sprintf("%s:/input", filepath.Join(d.rootDir, "fixtures", "v3")),
		os.Getenv("CNB_BUILD_IMAGE"),
		"/lifecycle/detector",
		"-buildpacks",
		"/buildpacks",
		"-order",
		"/input/order.toml",
		"-group",
		"/workspace/group.toml",
		"-plan",
		"/workspace/plan.toml",
	)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	result := &DetectResult{}

	_, err := toml.DecodeFile(filepath.Join(d.workspaceDir, "group.toml"), &result.Group)
	if err != nil {
		return nil, err
	}

	_, err = toml.DecodeFile(filepath.Join(d.workspaceDir, "plan.toml"), &result.BuildPlan)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (d *Dagger) Build(appDir string) (*libbuildpackV3.LaunchMetadata, error) {
	cmd := exec.Command(
		"docker",
		"run",
		"--rm",
		"-v",
		fmt.Sprintf("%s:/workspace", d.workspaceDir),
		"-v",
		fmt.Sprintf("%s:/workspace/app", appDir),
		"-v",
		fmt.Sprintf("%s:/buildpacks/org.cloudfoundry.buildpacks.nodejs/latest", d.buildpackDir),
		"-v",
		fmt.Sprintf("%s:/buildpacks/org.cloudfoundry.buildpacks.nodejs/1.6.32", d.buildpackDir),
		"-v",
		fmt.Sprintf("%s:/input", filepath.Join(d.rootDir, "fixtures", "v3")),
		os.Getenv("CNB_BUILD_IMAGE"),
		"/lifecycle/builder",
		"-buildpacks",
		"/buildpacks",
		"-group",
		"/input/group.toml",
		"-plan",
		"/input/plan.toml",
	)

	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	launch := &libbuildpackV3.LaunchMetadata{}
	_, err := toml.DecodeFile(filepath.Join(d.workspaceDir, "org.cloudfoundry.buildpacks.nodejs", "launch.toml"), &launch)
	if err != nil {
		return nil, err
	}

	return launch, nil
}

func copyFile(from, to string) error {
	source, err := os.Open(from)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(to)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

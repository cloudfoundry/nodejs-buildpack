package dagger

import (
	"bytes"
	"fmt"
	"github.com/BurntSushi/toml"
	libbuildpackV3 "github.com/buildpack/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/cutlass"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Dagger struct {
	rootDir, workspaceDir, buildpackDir, packDir string
}

func NewDagger(rootDir string) (*Dagger, error) {
	buildpackDir, err := ioutil.TempDir("/tmp", "buildpack")
	if err != nil {
		return nil, err
	}

	if err := os.Chmod(buildpackDir, 0755); err != nil {
		return nil, err
	}

	workspaceDir, err := ioutil.TempDir("/tmp", "workspace")
	if err != nil {
		return nil, err
	}

	if err := os.Chmod(workspaceDir, os.ModePerm); err != nil {
		return nil, err
	}

	packDir, err := ioutil.TempDir("/tmp", "pack")
	if err != nil {
		return nil, err
	}

	return &Dagger{
		rootDir:      rootDir,
		workspaceDir: workspaceDir,
		buildpackDir: buildpackDir,
		packDir:      packDir,
	}, nil
}

func (d *Dagger) Destroy() {
	os.RemoveAll(d.workspaceDir)
	d.workspaceDir = ""

	os.RemoveAll(d.buildpackDir)
	d.buildpackDir = ""

	os.RemoveAll(d.packDir)
	d.packDir = ""
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
			Id      string
			Version string
		}
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

type Layer struct {
	Metadata struct {
		Version string
	}
	Root string
}

type BuildResult struct {
	LaunchMetadata libbuildpackV3.LaunchMetadata
	Layer          Layer
}

func (d *Dagger) Build(appDir string) (*BuildResult, error) {
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

	rootDir := filepath.Join(d.workspaceDir, "org.cloudfoundry.buildpacks.nodejs")

	launchMetadata := libbuildpackV3.LaunchMetadata{}
	_, err := toml.DecodeFile(filepath.Join(rootDir, "launch.toml"), &launchMetadata)
	if err != nil {
		return nil, err
	}

	nodeLayer := Layer{Root: rootDir}
	_, err = toml.DecodeFile(filepath.Join(nodeLayer.Root, "node.toml"), &nodeLayer.Metadata)
	if err != nil {
		return nil, err
	}

	return &BuildResult{
		LaunchMetadata: launchMetadata,
		Layer:          nodeLayer,
	}, nil
}

func (d *Dagger) Pack(appDir string) (*App, error) {
	// TODO : replace the following with pack create-builder when it is ready
	const originalImage = "cnb-pack-builder"

	cmd := exec.Command("pack", "create-builder", originalImage, "-b", filepath.Join(d.rootDir, "fixtures", "v3", "builder.toml"))
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	const builderImage = "cnb-acceptance-builder"
	cmd = exec.Command("docker", "build", filepath.Join(d.rootDir, "fixtures", "v3"), "-t", builderImage)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	// TODO : remove above the above when pack create-builder works

	appImageName := cutlass.RandStringRunes(16)
	cmd = exec.Command("pack", "build", appImageName, "--builder", builderImage)
	cmd.Dir = appDir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return &App{imageName: appImageName}, nil
}

type App struct {
	imageName   string
	containerId string
	port        string
}

func (a *App) Start() error {
	buf := &bytes.Buffer{}

	cmd := exec.Command("docker", "run", "-d", "-P", a.imageName)
	cmd.Stdout = buf
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	a.containerId = buf.String()[:12]

	// TODO : implement a timer that checks health and bails out after X tries
	// but for now lets just sleep :)
	// cmd = exec.Command("docker", "inspect", "-f", "{{.State.Health.Status}}", a.containerId)
	fmt.Fprintf(os.Stderr, "Waiting for container to become healthy...")
	time.Sleep(35 * time.Second)

	cmd = exec.Command("docker", "container", "port", a.containerId)
	cmd.Stdout = buf
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	a.port = strings.TrimSpace(strings.Split(buf.String(), ":")[1])

	return nil
}

func (a *App) Destroy() error {
	if a.containerId == "" {
		return nil
	}

	cmd := exec.Command("docker", "stop", a.containerId)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	a.containerId = ""
	a.port = ""

	return nil
}

func (a *App) HTTPGet(path string) error {
	resp, err := http.Get("http://localhost:" + a.port + path)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("received bad response from application")
	}

	return nil
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

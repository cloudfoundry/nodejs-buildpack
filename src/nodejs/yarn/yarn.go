package yarn

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

type Command interface {
	Execute(dir string, stdout io.Writer, stderr io.Writer, program string, args ...string) error
	Run(cmd *exec.Cmd) error
}

type Yarn struct {
	Command Command
	Log     *libbuildpack.Logger
}

func (y *Yarn) Build(buildDir, cacheDir string) error {
	y.Log.Info("Installing node modules (yarn.lock)")

	err := y.doBuild(buildDir, cacheDir)

	if err != nil {
		return err
	}

	return nil
}

func (y *Yarn) Rebuild(buildDir, cacheDir string) error {
	y.Log.Info("Rebuilding native dependencies")

	err := y.doBuild(buildDir, cacheDir)

	if err != nil {
		return err
	}

	return nil
}

func (y *Yarn) doBuild(buildDir, cacheDir string) error {
	yarnVersion, err := y.getYarnVersion(buildDir)
	if err != nil {
		return err
	}

	if strings.HasPrefix(yarnVersion, "1") {
		return y.doBuildClassic(buildDir, cacheDir)
	}

	return y.doBuildBerry(buildDir)
}

func (y *Yarn) getYarnVersion(buildDir string) (string, error) {
	cmd := exec.Command("yarn", "--version")
	cmd.Dir = buildDir
	//cmd.Stdout = y.Log.Output()
	//cmd.Stderr = y.Log.Output()
	cmd.Env = append(os.Environ(), "npm_config_nodedir="+os.Getenv("NODE_HOME"))

	versionOutput, err := cmd.Output()
	if err != nil {
		return "", err
	}
	yarnVersion := strings.TrimSpace(string(versionOutput))
	return yarnVersion, nil
}

func (y *Yarn) isYarnLocalCacheEnabled(buildDir string) (bool, error) {
	cmd := exec.Command("yarn", "config", "get", "enableGlobalCache")
	cmd.Dir = buildDir
	//cmd.Stdout = y.Log.Output()
	//cmd.Stderr = y.Log.Output()
	cmd.Env = append(os.Environ(), "npm_config_nodedir="+os.Getenv("NODE_HOME"))

	cacheStrategyOutput, err := cmd.Output()
	if err != nil {
		return false, err
	}

	yarnCacheStrategy := strings.TrimSpace(string(cacheStrategyOutput))

	if yarnCacheStrategy == "false" {
		return true, nil
	}

	return false, nil
}

func (y *Yarn) doBuildClassic(buildDir, cacheDir string) error {
	// Start by informing users that Yarn v1 is deprecated and they should upgrade to Yarn v4 (Berry)
	y.Log.Protip("Yarn v1 is deprecated and has been replaced with Yarn Berry by the Yarn organisation. Please upgrade to Yarn v4 (Berry) to avoid future deprecation warnings.", "https://yarnpkg.com/migration/guide")

	offline, err := libbuildpack.FileExists(filepath.Join(buildDir, "npm-packages-offline-cache"))
	if err != nil {
		return err
	}

	installArgs := []string{"install", "--pure-lockfile", "--ignore-engines", "--cache-folder", filepath.Join(cacheDir, ".cache/yarn"), "--check-files"}

	yarnConfig := map[string]string{}
	if offline {
		yarnOfflineMirror := filepath.Join(buildDir, "npm-packages-offline-cache")
		y.Log.Info("Found yarn mirror directory %s", yarnOfflineMirror)
		y.Log.Info("Running yarn in offline mode")

		installArgs = append(installArgs, "--offline")

		yarnConfig["yarn-offline-mirror"] = yarnOfflineMirror
		yarnConfig["yarn-offline-mirror-pruning"] = "false"
	} else {
		y.Log.Info("Running yarn in online mode")
		y.Log.Info("To run yarn in offline mode, see: https://classic.yarnpkg.com/blog/2016/11/24/offline-mirror/")

		yarnConfig["yarn-offline-mirror"] = filepath.Join(cacheDir, "npm-packages-offline-cache")
		yarnConfig["yarn-offline-mirror-pruning"] = "true"
	}

	for k, v := range yarnConfig {
		cmd := exec.Command("yarn", "config", "set", k, v)
		cmd.Dir = buildDir
		cmd.Stdout = io.Discard
		cmd.Stderr = os.Stderr
		if err := y.Command.Run(cmd); err != nil {
			return err
		}
	}

	cmd := exec.Command("yarn", installArgs...)
	cmd.Dir = buildDir
	//cmd.Stdout = y.Log.Output()
	//cmd.Stderr = y.Log.Output()
	cmd.Env = append(os.Environ(), "npm_config_nodedir="+os.Getenv("NODE_HOME"))
	if err := y.Command.Run(cmd); err != nil {
		return err
	}

	err = os.RemoveAll(filepath.Join(buildDir, "npm-packages-offline-cache"))
	if err != nil {
		panic(err)
	}

	return nil
}

func (y *Yarn) doBuildBerry(buildDir string) error {
	usesLocalCache, err := y.isYarnLocalCacheEnabled(buildDir)
	if err != nil {
		return err
	}

	installArgs := []string{"install", "--immutable"}

	if usesLocalCache {
		installArgs = append(installArgs, "--immutable-cache")
	}

	cmd := exec.Command("yarn", installArgs...)
	cmd.Dir = buildDir
	//cmd.Stdout = y.Log.Output()
	//cmd.Stderr = y.Log.Output()
	cmd.Env = append(os.Environ(), "npm_config_nodedir="+os.Getenv("NODE_HOME"))
	if err := y.Command.Run(cmd); err != nil {
		return err
	}

	err = os.RemoveAll(filepath.Join(buildDir, ".yarn/cache"))
	if err != nil {
		panic(err)
	}

	return nil
}

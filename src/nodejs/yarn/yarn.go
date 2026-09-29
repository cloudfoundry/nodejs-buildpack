package yarn

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"

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

func (y *Yarn) Build(buildDir, cacheDir string, isBerry bool) error {
	if isBerry {
		return y.buildBerry(buildDir)
	}
	return y.buildClassic(buildDir, cacheDir)
}

// buildBerry installs dependencies using the Yarn Berry (2.x/3.x/4.x) CLI.
//
// Berry's CLI is not backwards compatible with Classic's flags: there is no
// "yarn config set", "--pure-lockfile", "--ignore-engines", or
// "--cache-folder" equivalent. Berry manages its own cache/linker behavior
// via the project's own .yarnrc.yml, which is left untouched here. The
// closest equivalent to Classic's "--pure-lockfile" (fail rather than
// silently update the lockfile) is Berry's "--immutable" flag.
func (y *Yarn) buildBerry(buildDir string) error {
	y.Log.Info("Installing node modules (yarn.lock) [yarn berry]")

	cmd := exec.Command("yarn", "install", "--immutable")
	cmd.Dir = buildDir
	cmd.Stdout = y.Log.Output()
	cmd.Stderr = y.Log.Output()
	cmd.Env = append(os.Environ(), "npm_config_nodedir="+os.Getenv("NODE_HOME"))
	return y.Command.Run(cmd)
}

func (y *Yarn) buildClassic(buildDir, cacheDir string) error {
	y.Log.Info("Installing node modules (yarn.lock)")

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
		y.Log.Info("To run yarn in offline mode, see: https://yarnpkg.com/blog/2016/11/24/offline-mirror")

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
	cmd.Stdout = y.Log.Output()
	cmd.Stderr = y.Log.Output()
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

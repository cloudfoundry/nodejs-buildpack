package yarn

import (
	"io"
	"io/ioutil"
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

func (y *Yarn) Build(buildDir, cacheDir string) error {
	y.Log.Info("Installing node modules (yarn.lock)")

	offline, err := libbuildpack.FileExists(filepath.Join(buildDir, "npm-packages-offline-cache"))
	if err != nil {
		return err
	}

	installArgs := []string{"install", "--pure-lockfile", "--ignore-engines", "--cache-folder", filepath.Join(cacheDir, ".cache/yarn"), "--modules-folder", filepath.Join(buildDir, "node_modules")}
	checkArgs := []string{"check"}

	yarnConfig := map[string]string{}
	if offline {
		yarnOfflineMirror := filepath.Join(buildDir, "npm-packages-offline-cache")
		y.Log.Info("Found yarn mirror directory %s", yarnOfflineMirror)
		y.Log.Info("Running yarn in offline mode")

		installArgs = append(installArgs, "--offline")
		checkArgs = append(checkArgs, "--offline")

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
		cmd.Stdout = ioutil.Discard
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

	if err := y.Command.Execute(buildDir, ioutil.Discard, os.Stderr, "yarn", checkArgs...); err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			return err
		}
		y.Log.Warning("yarn.lock is outdated")
	} else {
		y.Log.Info("yarn.lock and package.json match")
	}

	return nil
}

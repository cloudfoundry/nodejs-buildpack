package pnpm

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

type PNPM struct {
	Command Command
	Log     *libbuildpack.Logger
}

func (p *PNPM) Build(buildDir, cacheDir string) error {
	p.Log.Info("Installing node modules (pnpm-lock.yaml)")

	offline, err := libbuildpack.FileExists(filepath.Join(buildDir, ".pnpm-store"))
	if err != nil {
		return err
	}

	pnpmCacheDir := filepath.Join(cacheDir, ".pnpm-store")
	installArgs := []string{"install", "--no-frozen-lockfile"}

	if offline {
		pnpmStore := filepath.Join(buildDir, ".pnpm-store")
		p.Log.Info("Found pnpm store directory %s", pnpmStore)
		p.Log.Info("Running pnpm in offline mode")
		installArgs = append(installArgs, "--offline")
	} else {
		p.Log.Info("Running pnpm in online mode")
	}

	cmd := exec.Command("pnpm", installArgs...)
	cmd.Dir = buildDir
	cmd.Stdout = p.Log.Output()
	cmd.Stderr = p.Log.Output()
	cmd.Env = append(os.Environ(), "npm_config_nodedir="+os.Getenv("NODE_HOME"))

	// Set pnpm store location for caching between builds
	if !offline {
		cmd.Env = append(cmd.Env, "PNPM_HOME="+pnpmCacheDir)
	}

	if err := p.Command.Run(cmd); err != nil {
		return err
	}

	return nil
}

package pnpm

import (
	"io"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
)

type Command interface {
	Execute(dir string, stdout io.Writer, stderr io.Writer, program string, args ...string) error
}

type PNPM struct {
	Command Command
	Log     *libbuildpack.Logger
}

func (p *PNPM) Build(buildDir, cacheDir string) error {
	p.Log.Info("Installing node modules (pnpm-lock.yaml)")

	storeDir := filepath.Join(cacheDir, ".pnpm-store")
	p.Log.Info("Using pnpm store directory: %s", storeDir)

	if err := p.Command.Execute(buildDir, io.Discard, os.Stderr, "pnpm", "config", "set", "store-dir", storeDir); err != nil {
		return err
	}

	installArgs := []string{"install", "--frozen-lockfile"}

	if os.Getenv("NPM_CONFIG_PRODUCTION") == "true" {
		p.Log.Info("NPM_CONFIG_PRODUCTION is true, installing only production dependencies")
		installArgs = append(installArgs, "--prod")
	}

	vendoredStore := filepath.Join(buildDir, ".pnpm-store")
	if exists, err := libbuildpack.FileExists(vendoredStore); err == nil && exists {
		p.Log.Info("Found vendored pnpm store at %s", vendoredStore)
		p.Log.Info("Running pnpm in offline mode")
		installArgs = append(installArgs, "--offline")

		if err := p.Command.Execute(buildDir, io.Discard, os.Stderr, "pnpm", "config", "set", "store-dir", vendoredStore); err != nil {
			return err
		}
	}

	return p.Command.Execute(buildDir, p.Log.Output(), p.Log.Output(), "pnpm", installArgs...)
}

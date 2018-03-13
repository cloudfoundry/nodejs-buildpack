package npm

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

type Command interface {
	Execute(dir string, stdout io.Writer, stderr io.Writer, program string, args ...string) error
}

type NPM struct {
	Command Command
	Log     *libbuildpack.Logger
}

func (n *NPM) Build(pkgDir, cacheDir string) error {
	doBuild, source, err := n.doBuild(pkgDir)
	if err != nil {
		return err
	}
	if !doBuild {
		return nil
	}

	n.Log.Info("Installing node modules (%s)", source)
	npmArgs := []string{"install", "--unsafe-perm", "--userconfig", filepath.Join(pkgDir, ".npmrc"), "--cache", filepath.Join(cacheDir, ".npm")}
	return n.Command.Execute(pkgDir, n.Log.Output(), n.Log.Output(), "npm", npmArgs...)
}

func (n *NPM) Rebuild(pkgDir string) error {
	doBuild, source, err := n.doBuild(pkgDir)
	if err != nil {
		return err
	}
	if !doBuild {
		return nil
	}

	n.Log.Info("Rebuilding any native modules")
	if err := n.Command.Execute(pkgDir, n.Log.Output(), n.Log.Output(), "npm", "rebuild", "--nodedir="+os.Getenv("NODE_HOME")); err != nil {
		return err
	}

	n.Log.Info("Installing any new modules (%s)", source)
	npmArgs := []string{"install", "--unsafe-perm", "--userconfig", filepath.Join(pkgDir, ".npmrc")}
	return n.Command.Execute(pkgDir, n.Log.Output(), n.Log.Output(), "npm", npmArgs...)
}

func (n *NPM) doBuild(pkgDir string) (bool, string, error) {
	pkgExists, err := libbuildpack.FileExists(filepath.Join(pkgDir, "package.json"))
	if err != nil {
		return false, "", err
	}

	if !pkgExists {
		n.Log.Info("Skipping (no package.json)")
		return false, "", nil
	}

	files := []string{"package.json"}
	for _, filename := range []string{"package-lock.json", "npm-shrinkwrap.json"} {
		if found, err := libbuildpack.FileExists(filepath.Join(pkgDir, filename)); err != nil {
			return false, "", err
		} else if found {
			files = append(files, filename)
		}
	}

	return true, strings.Join(files, " + "), nil
}

package checksum

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const separator = string(filepath.Separator)

type Checksum struct {
	dir   string
	start time.Time
}

func New(dir string) *Checksum {
	return &Checksum{
		dir:   dir,
		start: time.Now(),
	}
}

func Do(dir string, debug func(format string, args ...interface{}), exec func() error) error {
	checksum := New(dir)
	if sum, err := checksum.calc(); err == nil {
		debug("Checksum Before (%s): %s", dir, sum)
	}

	err := exec()
	if err != nil {
		return err
	}

	if sum, err := checksum.calc(); err == nil {
		debug("Checksum After (%s): %s", dir, sum)
	}

	var changedFiles []string
	err = filepath.Walk(checksum.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.ModTime().After(checksum.start) {
			relativePath, err := filepath.Rel(checksum.dir, path)
			if err != nil {
				return err
			}

			if parts := strings.Split(relativePath, separator); parts[0] != ".cloudfoundry" {
				changedFiles = append(changedFiles, strings.Join([]string{".", relativePath}, separator))
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	if len(changedFiles) > 0 {
		debug("Below files changed:")
		for _, file := range changedFiles {
			debug(file)
		}
	}

	return nil
}

func (c *Checksum) calc() (string, error) {
	h := md5.New()
	err := filepath.Walk(c.dir, func(path string, info os.FileInfo, err error) error {
		if info.Mode().IsRegular() {
			relpath, err := filepath.Rel(c.dir, path)
			if strings.HasPrefix(relpath, ".cloudfoundry/") {
				return nil
			}
			if err != nil {
				return err
			}
			if _, err := io.WriteString(h, relpath); err != nil {
				return err
			}
			if f, err := os.Open(path); err != nil {
				return err
			} else {
				if _, err := io.Copy(h, f); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

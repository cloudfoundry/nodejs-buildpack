/*
 * Copyright 2018 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package libjavabuildpack

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// CopyFile copies source file to destFile, creating all intermediate directories in destFile
func CopyFile(source, destFile string) error {
	fh, err := os.Open(source)
	if err != nil {
		return err
	}

	fileInfo, err := fh.Stat()
	if err != nil {
		return err
	}

	defer fh.Close()

	return WriteToFile(fh, destFile, fileInfo.Mode())
}

// ExtractTarGz extracts tarfile to destDir
func ExtractTarGz(tarFile, destDir string, stripComponents int) error {
	file, err := os.Open(tarFile)
	if err != nil {
		return err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gz.Close()
	return extractTar(gz, destDir, stripComponents)
}

// ExtractZip extracts zipfile to destDir
func ExtractZip(zipfile, destDir string, stripComponents int) error {
	r, err := zip.OpenReader(zipfile)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		path := filepath.Join(append([]string{destDir}, strings.Split(f.Name, string(filepath.Separator))[stripComponents:]...)...)

		rc, err := f.Open()
		if err != nil {
			return err
		}

		if f.FileInfo().IsDir() {
			err = os.MkdirAll(path, 0755)
		} else {
			err = WriteToFile(rc, path, f.Mode())
		}

		rc.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

// FileExists returns true if a file exists, otherwise false.
func FileExists(file string) (bool, error) {
	_, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// FindRoot returns the location of the root of the current buildpack.
func FindRoot() (string, error) {
	exec, err := osArgs(0)
	if err != nil {
		return "", err
	}

	dir, err := filepath.Abs(path.Dir(exec))
	if err != nil {
		return "", err
	}

	for {
		if dir == "/" {
			return "", fmt.Errorf("could not find buildpack.toml in the directory hierarchy")
		}

		f := filepath.Join(dir, "buildpack.toml")
		if exist, err := FileExists(f); err != nil {
			return "", err
		} else if exist {
			return dir, nil
		}

		dir, err = filepath.Abs(filepath.Join(dir, ".."))
		if err != nil {
			return "", err
		}
	}
}

// FromTomlFile decodes a TOML file into a struct.
func FromTomlFile(file string, v interface{}) error {
	_, err := toml.DecodeFile(file, v)
	return err
}

// WriteToFile writes the contents of an io.Reader to a file.
func WriteToFile(source io.Reader, destFile string, mode os.FileMode) error {
	err := os.MkdirAll(filepath.Dir(destFile), 0755)
	if err != nil {
		return err
	}

	fh, err := os.OpenFile(destFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer fh.Close()

	_, err = io.Copy(fh, source)
	if err != nil {
		return err
	}

	return nil
}

func extractTar(src io.Reader, destDir string, stripComponents int) error {
	tr := tar.NewReader(src)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}

		path := filepath.Join(append([]string{destDir}, strings.Split(hdr.Name, string(filepath.Separator))[stripComponents:]...)...)
		fi := hdr.FileInfo()

		if fi.IsDir() {
			err = os.MkdirAll(path, hdr.FileInfo().Mode())
		} else if fi.Mode()&os.ModeSymlink != 0 {
			target := hdr.Linkname
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			if err = os.Symlink(target, path); err != nil {
				return err
			}
		} else {
			err = WriteToFile(tr, path, hdr.FileInfo().Mode())
		}

		if err != nil {
			return err
		}
	}
	return nil
}

func osArgs(index int) (string, error) {
	if len(os.Args) < index+1 {
		return "", fmt.Errorf("incorrect number of command line arguments")
	}

	return os.Args[index], nil
}

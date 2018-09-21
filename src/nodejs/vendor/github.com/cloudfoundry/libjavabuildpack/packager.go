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
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/buildpack/libbuildpack"
)

// Packager is a root element for packaging up a buildpack
type Packager struct {
	Buildpack Buildpack
	Cache     Cache
	Logger    Logger
	Root      string
}

// Create creates a new buildpack package.
func (p Packager) Create() error {
	p.Logger.FirstLine("Packaging %s", p.Logger.PrettyVersion(p.Buildpack))

	if err := p.prePackage(); err != nil {
		return err
	}

	includedFiles, err := p.Buildpack.IncludeFiles()
	if err != nil {
		return err
	}

	dependencyFiles, err := p.cacheDependencies()
	if err != nil {
		return err
	}

	return p.createArchive(append(includedFiles, dependencyFiles...))
}

func (p Packager) addFile(out *tar.Writer, path string) error {
	p.Logger.SubsequentLine("Adding %s", path)

	file, err := os.Open(filepath.Join(p.Root, path))
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	header := new(tar.Header)
	header.Name = path
	header.Size = stat.Size()
	header.Mode = int64(stat.Mode())
	header.ModTime = stat.ModTime()

	if err := out.WriteHeader(header); err != nil {
		return err
	}

	_, err = io.Copy(out, file)
	return err
}

func (p Packager) archivePath() (string, error) {
	dir, err := osArgs(1)
	if err != nil {
		return "", err
	}

	info := p.Buildpack.Info

	return filepath.Join(dir, fmt.Sprintf("%s-v%s.tar.gz", info.ID, info.Version)), nil
}

func (p Packager) createArchive(files []string) error {
	archive, err := p.archivePath()
	if err != nil {
		return err
	}

	p.Logger.FirstLine("Creating archive %s", archive)

	file, err := os.OpenFile(archive, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	gw := gzip.NewWriter(file)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	for _, file := range files {
		if err := p.addFile(tw, file); err != nil {
			return err
		}
	}

	return nil
}

func (p Packager) defaultLogger() libbuildpack.Logger {
	var debug io.Writer

	if _, ok := os.LookupEnv("BP_DEBUG"); ok {
		debug = os.Stderr
	}

	return libbuildpack.NewLogger(debug, os.Stdout)
}

func (p Packager) cacheDependencies() ([]string, error) {
	var files []string

	deps, err := p.Buildpack.Dependencies()
	if err != nil {
		return nil, err
	}

	for _, dep := range deps {
		p.Logger.FirstLine("Caching %s", p.Logger.PrettyVersion(dep))

		layer := p.Cache.DownloadLayer(dep)

		a, err := layer.Artifact()
		if err != nil {
			return nil, err
		}

		artifact, err := filepath.Rel(p.Root, a)
		if err != nil {
			return nil, err
		}

		metadata, err := filepath.Rel(p.Root, layer.Metadata())
		if err != nil {
			return nil, err
		}

		files = append(files, artifact, metadata)
	}

	return files, nil
}

func (p Packager) prePackage() error {
	pp, ok := p.Buildpack.PrePackage()
	if !ok {
		return nil
	}

	cmd := exec.Command(pp)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = p.Root

	p.Logger.FirstLine("Pre-Package with %s", strings.Join(cmd.Args, " "))

	return cmd.Run()
}

// DefaultPackager creates a new Packager, using the executable to find the root of the buildpack.
func DefaultPackager() (Packager, error) {
	p := Packager{}

	root, err := FindRoot()
	if err != nil {
		return Packager{}, err
	}
	p.Root = root

	logger := p.defaultLogger()
	p.Logger = Logger{Logger: logger}

	buildpack, err := libbuildpack.DefaultBuildpack(logger)
	if err != nil {
		return Packager{}, err
	}
	p.Buildpack = Buildpack{Buildpack: buildpack}

	cache := libbuildpack.Cache{Root: filepath.Join(root, "cache"), Logger: logger}
	p.Cache = Cache{Cache: cache, Logger: p.Logger}

	return p, nil
}

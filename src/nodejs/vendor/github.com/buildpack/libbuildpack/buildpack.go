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

package libbuildpack

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/buildpack/libbuildpack/internal"
)

// Buildpack represents the metadata associated with a buildpack.
type Buildpack struct {
	// Info is information about the buildpack.
	Info BuildpackInfo `toml:"buildpack"`

	// Stacks is the collection of stacks that the buildpack supports.
	Stacks []BuildpackStack `toml:"stacks"`

	// Metadata is the additional metadata included in the buildpack.
	Metadata BuildpackMetadata `toml:"metadata"`

	// Logger is used to write debug and info to the console.
	Logger Logger
}

// String makes Buildpack satisfy the Stringer interface.
func (b Buildpack) String() string {
	return fmt.Sprintf("Buildpack{ Info: %s, Stacks: %s, Metadata: %s, Logger: %s }",
		b.Info, b.Stacks, b.Metadata, b.Logger)
}

// BuildpackInfo is information about the buildpack.
type BuildpackInfo struct {
	// ID is the globally unique identifier of the buildpack.
	ID string `toml:"id"`

	// Name is the human readable name of the buildpack.
	Name string `toml:"name"`

	// Version is the semver-compliant version of the buildpack.
	Version string `toml:"version"`
}

// String makes BuildpackInfo satisfy the Stringer interface.
func (b BuildpackInfo) String() string {
	return fmt.Sprintf("BuildpackInfo{ ID: %s, Name: %s, Version: %s }", b.ID, b.Name, b.Version)
}

// BuildpackMetadata is additional metadata included in the buildpack
type BuildpackMetadata map[string]interface{}

// BuildpackStack represents metadata about the stacks associated with the buildpack.
type BuildpackStack struct {
	// ID is the globally unique identifier of the stack.
	ID string `toml:"id"`

	// BuildImages are the suggested sources for stacks if the platform is unaware of the stack id.
	BuildImages []BuildImages `toml:"build-images"`

	// RunImages are the suggested sources for stacks if the platform is unaware of the stack id.
	RunImages []RunImages `toml:"run-images"`
}

// String makes BuildpackStack satisfy the Stringer interface.
func (b BuildpackStack) String() string {
	return fmt.Sprintf("BuildpackStack{ ID: %s, BuildImages: %s, RunImages: %s }", b.ID, b.BuildImages, b.RunImages)
}

// BuildImages is the build image source for a particular stack id.
type BuildImages string

// RunImages is the run image source for a particular stack id.
type RunImages string

// DefaultBuildpack creates a new instance of Buildpack extracting the contents of the buildpack.toml file in the root
// of the buildpack.
func DefaultBuildpack(logger Logger) (Buildpack, error) {
	f, err := findBuildpackToml()
	if err != nil {
		return Buildpack{}, err
	}

	in, err := os.Open(f)
	if err != nil {
		return Buildpack{}, err
	}
	defer in.Close()

	return NewBuildpack(in, logger)
}

// NewBuildpack creates a new instance of Buildpack from a specified io.Reader.  Returns an error if the contents of
// the reader are not valid TOML.
func NewBuildpack(in io.Reader, logger Logger) (Buildpack, error) {
	b := Buildpack{Logger: logger}

	if _, err := toml.DecodeReader(in, &b); err != nil {
		return Buildpack{}, err
	}

	logger.Debug("Buildpack: %s", b)
	return b, nil
}

func findBuildpackToml() (string, error) {
	exec, err := internal.OsArgs(0)
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
		if exist, err := internal.FileExists(f); err != nil {
			return "", err
		} else if exist {
			return f, nil
		}

		dir, err = filepath.Abs(filepath.Join(dir, ".."))
		if err != nil {
			return "", err
		}
	}
}

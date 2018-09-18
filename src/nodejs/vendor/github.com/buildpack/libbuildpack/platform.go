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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/buildpack/libbuildpack/internal"
)

// Platform represents the platform contributions for an application.
type Platform struct {
	// Root is the path to the root directory for the platform contributions.
	Root string

	// Envs is the collection of environment variables contributed by the platform.
	Envs EnvironmentVariables

	// Logger is used to write debug and info to the console.
	Logger Logger
}

// String makes Platform satisfy the Stringer interface.
func (p Platform) String() string {
	return fmt.Sprintf("Platform{ Root: %s, Envs: %s, Logger: %s }", p.Root, p.Envs, p.Logger)
}

func (p Platform) enumerateEnvs(logger Logger) (EnvironmentVariables, error) {
	files, err := filepath.Glob(filepath.Join(p.Root, "env", "*"))
	if err != nil {
		return nil, err
	}

	var e EnvironmentVariables

	for _, file := range files {
		e = append(e, EnvironmentVariable{filepath.Base(file), file, logger})
	}

	return e, nil
}

// EnvironmentVariables is a collection of EnvironmentVariable instances.
type EnvironmentVariables []EnvironmentVariable

// SetAll sets all of the environment variable content in the current process environment.
func (e EnvironmentVariables) SetAll() error {
	for _, ev := range e {
		if err := ev.Set(); err != nil {
			return err
		}
	}

	return nil
}

// EnvironmentVariable represents an environment variable provided by the platform.
type EnvironmentVariable struct {
	// Name is the name of the environment variable
	Name string

	file   string
	logger Logger
}

// String makes EnvironmentVariable satisfy the Stringer interface.
func (e EnvironmentVariable) String() string {
	return fmt.Sprintf("EnvironmentVariable{ Name: %s, file: %s, logger: %s }", e.Name, e.file, e.logger)
}

// Set sets the environment variable content in the current process environment.
func (e EnvironmentVariable) Set() error {
	value, err := e.value()
	if err != nil {
		return err
	}

	e.logger.Debug("Setting environment variable: %s <= %s", e.Name, value)
	return os.Setenv(e.Name, value)
}

func (e EnvironmentVariable) value() (string, error) {
	b, err := ioutil.ReadFile(e.file)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

// DefaultPlatform creates a new instance of Platform, extracting the Root path from os.Args[1].
func DefaultPlatform(logger Logger) (Platform, error) {
	root, err := internal.OsArgs(1)
	if err != nil {
		return Platform{}, err
	}

	return NewPlatform(root, logger)
}

// NewPlatform creates a new instance of Platform, configuring the Root path.
func NewPlatform(root string, logger Logger) (Platform, error) {
	p := Platform{Root: root, Logger: logger}

	if logger.IsDebugEnabled() {
		logger.Debug("Platform contents: %s", internal.DirectoryContents(root))
	}

	envs, err := p.enumerateEnvs(logger)
	if err != nil {
		return Platform{}, err
	}

	logger.Debug("Platform environment variables: %s", envs)
	p.Envs = envs

	return p, err
}

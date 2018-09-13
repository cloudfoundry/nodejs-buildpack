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
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Launch represents launch layers for an application.
type Launch struct {
	// Root is the path to the root directory for the layers.
	Root string

	logger Logger
}

// Layer creates a LaunchLayer with a specified name.
func (l Launch) Layer(name string) LaunchLayer {
	metadata := filepath.Join(l.Root, fmt.Sprintf("%s.toml", name))
	return LaunchLayer{filepath.Join(l.Root, name), l.logger, metadata}
}

// String makes Launch satisfy the Stringer interface.
func (l Launch) String() string {
	return fmt.Sprintf("Launch{ Root: %s, logger: %s }", l.Root, l.logger)
}

// WriteMetadata writes Launch metadata to the filesystem.
func (l Launch) WriteMetadata(metadata LaunchMetadata) error {
	m, err := toTomlString(metadata)
	if err != nil {
		return err
	}

	f := filepath.Join(l.Root, "launch.toml")

	l.logger.Debug("Writing launch metadata: %s <= %s", f, m)
	return writeToFile(strings.NewReader(m), f, 0644)
}

// LaunchLayer represents a launch layer for an application.
type LaunchLayer struct {
	// Root is the path to the root directory for the launch layer.
	Root string

	logger   Logger
	metadata string
}

// String makes LaunchLayer satisfy the Stringer interface.
func (l LaunchLayer) String() string {
	return fmt.Sprintf("LaunchLayer{ Root: %s, logger: %s }", l.Root, l.logger)
}

// ReadMetadata reads arbitrary launch layer metadata from the filesystem.
func (l LaunchLayer) ReadMetadata(v interface{}) error {
	exists, err := fileExists(l.metadata)
	if err != nil {
		return err
	}

	if !exists {
		l.logger.Debug("Metadata %s does not exist", l.metadata)
		return nil
	}

	_, err = toml.DecodeFile(l.metadata, v)
	if err != nil {
		return err
	}

	l.logger.Debug("Reading layer metadata: %s <= %v", l.metadata, v)
	return nil
}

// WriteMetadata writes arbitrary launch layer metadata to the filesystem.
func (l LaunchLayer) WriteMetadata(metadata interface{}) error {
	m, err := toTomlString(metadata)
	if err != nil {
		return err
	}

	l.logger.Debug("Writing layer metadata: %s <= %s", l.metadata, m)
	return writeToFile(strings.NewReader(m), l.metadata, 0644)
}

// WriteProfile writes a file to profile.d with this value.
func (l LaunchLayer) WriteProfile(file string, format string, args ...interface{}) error {
	f := filepath.Join(l.Root, "profile.d", file)
	v := fmt.Sprintf(format, args...)

	l.logger.Debug("Writing profile: %s <= %s", f, v)

	return writeToFile(strings.NewReader(v), f, 0644)
}

// LaunchMetadata represents metadata about the Launch.
type LaunchMetadata struct {
	// Processes is a collection of processes.
	Processes Processes `toml:"processes"`
}

// String makes LaunchMetadata satisfy the Stringer interface.
func (l LaunchMetadata) String() string {
	return fmt.Sprintf("LaunchMetadata{ Processes: %s }", l.Processes)
}

// Processes is a collection of Process instances.
type Processes []Process

// Process represents metadata about a type of command that can be run.
type Process struct {
	// Type is the type of the process.
	Type string `toml:"type"`

	// Command is the command of the process.
	Command string `toml:"command"`
}

// String makes Process satisfy the Stringer interface.
func (p Process) String() string {
	return fmt.Sprintf("Process{ Type: %s, Command: %s }", p.Type, p.Command)
}

// DefaultLaunch creates a new instance of Launch, extracting the Root path from os.Args[3].
func DefaultLaunch(logger Logger) (Launch, error) {
	root, err := osArgs(3)
	if err != nil {
		return Launch{}, err
	}

	logger.Debug("Launch root: %s", root)

	return Launch{root, logger}, nil
}

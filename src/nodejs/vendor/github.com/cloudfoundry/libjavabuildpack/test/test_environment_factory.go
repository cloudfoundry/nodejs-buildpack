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

package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudfoundry/libjavabuildpack"
	"github.com/cloudfoundry/libjavabuildpack/internal"
)

// EnvironmentFactory is a factory for creating a test environment to test detect and build functionality.
type EnvironmentFactory struct {
	// Application is the root of the application
	Application string

	// Console is the Console of the application.
	Console Console

	// ExitStatus is the exit status code of the application.
	ExitStatus *int

	defers []func()
}

// Restore restores the original environment before testing.
//
// defer f.Restore()
func (f *EnvironmentFactory) Restore() {
	for _, d := range f.defers {
		d()
	}
}

// NewEnvironmentFactory creates a new instance of EnvironmentFactory.
func NewEnvironmentFactory(t *testing.T) EnvironmentFactory {
	t.Helper()
	f := EnvironmentFactory{}

	root := ScratchDir(t, "test-environment-factory")

	appRoot := filepath.Join(root, "app")
	if err := os.MkdirAll(appRoot, 0755); err != nil {
		t.Fatal(err)
	}
	f.Application = appRoot

	d := ReplaceWorkingDirectory(t, appRoot)
	f.defers = append(f.defers, d)

	f.Console, d = ReplaceConsole(t)
	f.defers = append(f.defers, d)

	d = ReplaceEnv(t, "PACK_STACK_ID", "test-stack")
	f.defers = append(f.defers, d)

	buildpackRoot := filepath.Join(root, "buildpack")
	b, err := internal.ToTomlString(libjavabuildpack.Buildpack{})
	if err != nil {
		t.Fatal(err)
	}
	if err := libjavabuildpack.WriteToFile(strings.NewReader(b), filepath.Join(buildpackRoot, "buildpack.toml"), 0644); err != nil {
		t.Fatal(err)
	}

	d = ReplaceArgs(t, filepath.Join(buildpackRoot, "bin", "test"), filepath.Join(root, "platform"),
		filepath.Join(root, "cache"), filepath.Join(root, "launch"))
	f.defers = append(f.defers, d)

	f.ExitStatus, d = CaptureExitStatus(t)
	f.defers = append(f.defers, d)

	return f
}

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

package libbuildpack_test

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/buildpack/libbuildpack"
	"github.com/buildpack/libbuildpack/internal"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestLaunch(t *testing.T) {
	spec.Run(t, "Launch", testLaunch, spec.Report(report.Terminal{}))
}

func testLaunch(t *testing.T, when spec.G, it spec.S) {

	logger := libbuildpack.Logger{}

	type metadata struct {
		Alpha string
		Bravo int
	}

	it("extracts root from os.Args[3]", func() {
		defer internal.ReplaceArgs(t, "", "", "", "launch-root")()

		launch, err := libbuildpack.DefaultLaunch(logger)
		if err != nil {
			t.Fatal(err)
		}

		if launch.Root != "launch-root" {
			t.Errorf("Launch.Root = %s, expected = launch-root", launch.Root)
		}
	})

	it("creates a launch layer with root based on its name", func() {
		launch := libbuildpack.Launch{Root: "test-root"}
		layer := launch.Layer("test-layer")

		if layer.Root != "test-root/test-layer" {
			t.Errorf("LaunchLayer.Root = %s, expected test-root/test-layer", layer.Root)
		}
	})

	it("writes a profile file", func() {
		root := internal.ScratchDir(t, "launch")
		layer := libbuildpack.LaunchLayer{Root: root}

		if err := layer.WriteProfile("test-name", "%s-%d", "test-string", 1); err != nil {
			t.Fatal(err)
		}

		internal.BeFileLike(t, filepath.Join(root, "profile.d", "test-name"), 0644, "test-string-1")
	})

	it("writes launch metadata", func() {
		root := internal.ScratchDir(t, "launch")
		launch := libbuildpack.Launch{Root: root}

		lm := libbuildpack.LaunchMetadata{
			Processes: libbuildpack.Processes{
				libbuildpack.Process{Type: "web", Command: "command-1"},
				libbuildpack.Process{Type: "task", Command: "command-2"},
			},
		}

		if err := launch.WriteMetadata(lm); err != nil {
			t.Fatal(err)
		}

		internal.BeFileLike(t, filepath.Join(root, "launch.toml"), 0644, `[[processes]]
  type = "web"
  command = "command-1"

[[processes]]
  type = "task"
  command = "command-2"
`)
	})

	it("reads layer content metadata", func() {
		root := internal.ScratchDir(t, "launch")
		launch := libbuildpack.Launch{Root: root}
		layer := launch.Layer("test-layer")

		internal.WriteToFile(strings.NewReader(`Alpha = "test-value"
Bravo = 1
`), filepath.Join(root, "test-layer.toml"), 0644)

		var actual metadata
		if err := layer.ReadMetadata(&actual); err != nil {
			t.Fatal(err)
		}

		expected := metadata{"test-value", 1}

		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("metadata = %v, wanted %v", actual, expected)
		}
	})

	it("does not read layer content metadata if it does not exist", func() {
		root := internal.ScratchDir(t, "launch")
		launch := libbuildpack.Launch{Root: root}
		layer := launch.Layer("test-layer")

		var actual metadata
		if err := layer.ReadMetadata(&actual); err != nil {
			t.Fatal(err)
		}

		expected := metadata{}

		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("metadata = %v, wanted %v", actual, expected)
		}
	})

	it("writes layer content metadata", func() {
		root := internal.ScratchDir(t, "launch")
		launch := libbuildpack.Launch{Root: root}
		layer := launch.Layer("test-layer")

		if err := layer.WriteMetadata(metadata{"test-value", 1}); err != nil {
			t.Fatal(err)
		}

		internal.BeFileLike(t, filepath.Join(root, "test-layer.toml"), 0644, `Alpha = "test-value"
Bravo = 1
`)
	})
}

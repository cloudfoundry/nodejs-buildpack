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

func TestBuild(t *testing.T) {
	spec.Run(t, "Build", testBuild, spec.Report(report.Terminal{}))
}

func testBuild(t *testing.T, when spec.G, it spec.S) {

	it("contains default values", func() {
		root := internal.ScratchDir(t, "detect")
		defer internal.ReplaceWorkingDirectory(t, root)()
		defer internal.ReplaceEnv(t, "PACK_STACK_ID", "test-stack")()

		console, d := internal.ReplaceConsole(t)
		defer d()

		console.In(t, `[alpha]
  version = "alpha-version"
  name = "alpha-name"

[bravo]
  name = "bravo-name"
`)

		in := strings.NewReader(`[buildpack]
id = "buildpack-id"
name = "buildpack-name"
version = "buildpack-version"

[[stacks]]
id = 'stack-id'
build-images = ["build-image-tag"]
run-images = ["run-image-tag"]

[metadata]
test-key = "test-value"
`)

		err := internal.WriteToFile(in, filepath.Join(root, "buildpack.toml"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		defer internal.ReplaceArgs(t, filepath.Join(root, "bin", "test"), root, root, root)()

		build, err := libbuildpack.DefaultBuild()
		if err != nil {
			t.Fatal(err)
		}

		if reflect.DeepEqual(build.Application, libbuildpack.Application{}) {
			t.Errorf("detect.Application should not be empty")
		}

		if reflect.DeepEqual(build.Buildpack, libbuildpack.Buildpack{}) {
			t.Errorf("detect.Buildpack should not be empty")
		}

		if reflect.DeepEqual(build.BuildPlan, libbuildpack.BuildPlan{}) {
			t.Errorf("detect.BuildPlan should not be empty")
		}

		if reflect.DeepEqual(build.Cache, libbuildpack.Cache{}) {
			t.Errorf("detect.Cache should not be empty")
		}

		if reflect.DeepEqual(build.Launch, libbuildpack.Launch{}) {
			t.Errorf("detect.Launch should not be empty")
		}

		if reflect.DeepEqual(build.Logger, libbuildpack.Logger{}) {
			t.Errorf("detect.Logger should not be empty")
		}

		if reflect.DeepEqual(build.Platform, libbuildpack.Platform{}) {
			t.Errorf("detect.Platform should not be empty")
		}

		if reflect.DeepEqual(build.Stack, libbuildpack.Stack("")) {
			t.Errorf("detect.Stack should not be empty")
		}
	})

	it("suppresses debug output", func() {
		root := internal.ScratchDir(t, "build")
		defer internal.ReplaceWorkingDirectory(t, root)()
		defer internal.ReplaceEnv(t, "PACK_STACK_ID", "test-stack")()

		c, d := internal.ReplaceConsole(t)
		defer d()
		c.In(t, "")

		err := internal.WriteToFile(strings.NewReader(""), filepath.Join(root, "buildpack.toml"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		defer internal.ReplaceArgs(t, filepath.Join(root, "bin", "test"), root, root, root)()

		build, err := libbuildpack.DefaultBuild()
		if err != nil {
			t.Fatal(err)
		}

		build.Logger.Debug("test-debug-output")
		build.Logger.Info("test-info-output")

		if strings.Contains(c.Err(t), "test-debug-output") {
			t.Errorf("stderr contained test-debug-output, expected not")
		}

		if !strings.Contains(c.Out(t), "test-info-output") {
			t.Errorf("stdout did not contain test-info-output, expected to")
		}
	})

	it("allows debug output if BP_DEBUG is set", func() {
		root := internal.ScratchDir(t, "build")
		defer internal.ReplaceWorkingDirectory(t, root)()
		defer internal.ReplaceEnv(t, "PACK_STACK_ID", "test-stack")()

		c, d := internal.ReplaceConsole(t)
		defer d()
		c.In(t, "")

		err := internal.WriteToFile(strings.NewReader(""), filepath.Join(root, "buildpack.toml"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		defer internal.ReplaceArgs(t, filepath.Join(root, "bin", "test"), root, root, root)()
		defer internal.ReplaceEnv(t, "BP_DEBUG", "")()

		build, err := libbuildpack.DefaultBuild()
		if err != nil {
			t.Fatal(err)
		}

		build.Logger.Debug("test-debug-output")
		build.Logger.Info("test-info-output")

		if !strings.Contains(c.Err(t), "test-debug-output") {
			t.Errorf("stderr did not contain test-debug-output, expected to")
		}

		if !strings.Contains(c.Out(t), "test-info-output") {
			t.Errorf("stdout did not contain test-info-output, expected to")
		}
	})

	it("returns 0 when successful", func() {
		root := internal.ScratchDir(t, "build")
		defer internal.ReplaceWorkingDirectory(t, root)()
		defer internal.ReplaceEnv(t, "PACK_STACK_ID", "test-stack")()

		c, d := internal.ReplaceConsole(t)
		defer d()
		c.In(t, "")

		err := internal.WriteToFile(strings.NewReader(""), filepath.Join(root, "buildpack.toml"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		defer internal.ReplaceArgs(t, filepath.Join(root, "bin", "test"), root, root, root)()

		build, err := libbuildpack.DefaultBuild()
		if err != nil {
			t.Fatal(err)
		}

		expected, d := internal.CaptureExitStatus(t)
		defer d()

		build.Success()

		if *expected != 0 {
			t.Errorf("os.Exit = %d, expected 0", *expected)
		}
	})

	it("returns code when failing", func() {
		root := internal.ScratchDir(t, "build")
		defer internal.ReplaceWorkingDirectory(t, root)()
		defer internal.ReplaceEnv(t, "PACK_STACK_ID", "test-stack")()

		c, d := internal.ReplaceConsole(t)
		defer d()
		c.In(t, "")

		err := internal.WriteToFile(strings.NewReader(""), filepath.Join(root, "buildpack.toml"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		defer internal.ReplaceArgs(t, filepath.Join(root, "bin", "test"), root, root, root)()

		build, err := libbuildpack.DefaultBuild()
		if err != nil {
			t.Fatal(err)
		}

		expected, d := internal.CaptureExitStatus(t)
		defer d()

		build.Failure(42)

		if *expected != 42 {
			t.Errorf("os.Exit = %d, expected 42", *expected)
		}
	})
}

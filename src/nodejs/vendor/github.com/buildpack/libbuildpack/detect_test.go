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

func TestDetect(t *testing.T) {
	spec.Run(t, "Detect", testDetect, spec.Report(report.Terminal{}))
}

func testDetect(t *testing.T, when spec.G, it spec.S) {

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

		defer internal.ReplaceArgs(t, filepath.Join(root, "bin", "test"))()

		detect, err := libbuildpack.DefaultDetect()
		if err != nil {
			t.Fatal(err)
		}

		if reflect.DeepEqual(detect.Application, libbuildpack.Application{}) {
			t.Errorf("detect.Application should not be empty")
		}

		if reflect.DeepEqual(detect.Buildpack, libbuildpack.Buildpack{}) {
			t.Errorf("detect.Buildpack should not be empty")
		}

		if reflect.DeepEqual(detect.BuildPlan, libbuildpack.BuildPlan{}) {
			t.Errorf("detect.BuildPlan should not be empty")
		}

		if reflect.DeepEqual(detect.Logger, libbuildpack.Logger{}) {
			t.Errorf("detect.Logger should not be empty")
		}

		if reflect.DeepEqual(detect.Stack, "") {
			t.Errorf("detect.Stack should not be empty")
		}
	})

	it("suppresses debug output", func() {
		root := internal.ScratchDir(t, "detect")
		defer internal.ReplaceWorkingDirectory(t, root)()
		defer internal.ReplaceEnv(t, "PACK_STACK_ID", "test-stack")()

		c, d := internal.ReplaceConsole(t)
		defer d()
		c.In(t, "")

		err := internal.WriteToFile(strings.NewReader(""), filepath.Join(root, "buildpack.toml"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		defer internal.ReplaceArgs(t, filepath.Join(root, "bin", "test"))()

		detect, err := libbuildpack.DefaultDetect()
		if err != nil {
			t.Fatal(err)
		}

		detect.Logger.Debug("test-debug-output")
		detect.Logger.Info("test-info-output")

		stderr := c.Err(t)

		if strings.Contains(stderr, "test-debug-output") {
			t.Errorf("stderr contained test-debug-output, expected not")
		}

		if !strings.Contains(stderr, "test-info-output") {
			t.Errorf("stderr did not contain test-info-output, expected to")
		}

		if c.Out(t) != "" {
			t.Errorf("stdout was not empty, expected empty")
		}
	})

	it("allows debug output if BP_DEBUG is set", func() {
		root := internal.ScratchDir(t, "detect")
		defer internal.ReplaceWorkingDirectory(t, root)()
		defer internal.ReplaceEnv(t, "PACK_STACK_ID", "test-stack")()

		c, d := internal.ReplaceConsole(t)
		defer d()
		c.In(t, "")

		err := internal.WriteToFile(strings.NewReader(""), filepath.Join(root, "buildpack.toml"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		defer internal.ReplaceArgs(t, filepath.Join(root, "bin", "test"))()
		defer internal.ReplaceEnv(t, "BP_DEBUG", "")()

		detect, err := libbuildpack.DefaultDetect()
		if err != nil {
			t.Fatal(err)
		}

		detect.Logger.Debug("test-debug-output")
		detect.Logger.Info("test-info-output")

		stderr := c.Err(t)

		if !strings.Contains(stderr, "test-debug-output") {
			t.Errorf("stderr did not contain test-debug-output, expected to")
		}

		if !strings.Contains(stderr, "test-info-output") {
			t.Errorf("stderr did not contain test-info-output, expected to")
		}

		if c.Out(t) != "" {
			t.Errorf("stdout was not empty, expected empty")
		}
	})

	it("returns code when erroring", func() {
		root := internal.ScratchDir(t, "detect")
		defer internal.ReplaceWorkingDirectory(t, root)()
		defer internal.ReplaceEnv(t, "PACK_STACK_ID", "test-stack")()

		c, d := internal.ReplaceConsole(t)
		defer d()
		c.In(t, "")

		err := internal.WriteToFile(strings.NewReader(""), filepath.Join(root, "buildpack.toml"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		defer internal.ReplaceArgs(t, filepath.Join(root, "bin", "test"))()

		detect, err := libbuildpack.DefaultDetect()
		if err != nil {
			t.Fatal(err)
		}

		actual, d := internal.CaptureExitStatus(t)
		defer d()

		detect.Error(42)

		if *actual != 42 {
			t.Errorf("os.Exit = %d, expected 42", *actual)
		}
	})

	it("returns 100 when failing", func() {
		root := internal.ScratchDir(t, "detect")
		defer internal.ReplaceWorkingDirectory(t, root)()
		defer internal.ReplaceEnv(t, "PACK_STACK_ID", "test-stack")()

		c, d := internal.ReplaceConsole(t)
		defer d()
		c.In(t, "")

		err := internal.WriteToFile(strings.NewReader(""), filepath.Join(root, "buildpack.toml"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		defer internal.ReplaceArgs(t, filepath.Join(root, "bin", "test"))()

		detect, err := libbuildpack.DefaultDetect()
		if err != nil {
			t.Fatal(err)
		}

		actual, d := internal.CaptureExitStatus(t)
		defer d()

		detect.Fail()

		if *actual != 100 {
			t.Errorf("os.Exit = %d, expected 100", *actual)
		}
	})

	it("returns 0 and BuildPlan when passing", func() {

		root := internal.ScratchDir(t, "detect")
		defer internal.ReplaceWorkingDirectory(t, root)()
		defer internal.ReplaceEnv(t, "PACK_STACK_ID", "test-stack")()

		c, d := internal.ReplaceConsole(t)
		defer d()
		c.In(t, "")

		err := internal.WriteToFile(strings.NewReader(""), filepath.Join(root, "buildpack.toml"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		defer internal.ReplaceArgs(t, filepath.Join(root, "bin", "test"))()

		detect, err := libbuildpack.DefaultDetect()
		if err != nil {
			t.Fatal(err)
		}

		actual, d := internal.CaptureExitStatus(t)
		defer d()

		detect.Pass(libbuildpack.BuildPlan{
			"alpha": libbuildpack.BuildPlanDependency{Provider: "test-provider", Version: "test-version"},
		})

		if *actual != 0 {
			t.Errorf("os.Exit = %d, expected 0", *actual)
		}

		stdout := c.Out(t)
		expectedStdout := `[alpha]
  provider = "test-provider"
  version = "test-version"
`

		if stdout != expectedStdout {
			t.Errorf("stdout = %s, expected %s", stdout, expectedStdout)
		}
	})
}

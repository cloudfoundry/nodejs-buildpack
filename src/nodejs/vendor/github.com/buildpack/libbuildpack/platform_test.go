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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buildpack/libbuildpack"
	"github.com/buildpack/libbuildpack/internal"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestPlatform(t *testing.T) {
	spec.Run(t, "Platform", testPlatform, spec.Report(report.Terminal{}))
}

func testPlatform(t *testing.T, when spec.G, it spec.S) {

	var logger = libbuildpack.NewLogger(nil, nil)

	it("returns the root of the platform", func() {
		root := internal.ScratchDir(t, "platform")

		platform, err := libbuildpack.NewPlatform(root, logger)
		if err != nil {
			t.Fatal(err)
		}

		if platform.Root != root {
			t.Errorf("Platform.Root = %s, wanted %s", platform.Root, root)
		}
	})

	it("extracts root from os.Args[1]", func() {
		root := internal.ScratchDir(t, "platform")
		defer internal.ReplaceArgs(t, "", root)()

		platform, err := libbuildpack.DefaultPlatform(logger)
		if err != nil {
			t.Fatal(err)
		}

		if platform.Root != root {
			t.Errorf("Platform.Root = %s, wanted %s", platform.Root, root)
		}
	})

	it("enumerates platform environment variables", func() {
		root := internal.ScratchDir(t, "platform")
		internal.WriteToFile(strings.NewReader("test-value"), filepath.Join(root, "env", "TEST_KEY"), 0644)

		platform, err := libbuildpack.NewPlatform(root, logger)
		if err != nil {
			t.Fatal(err)
		}

		if platform.Envs[0].Name != "TEST_KEY" {
			t.Errorf("Platform.Envs[0].Name = %s, expected TEST_KEY", platform.Envs[0])
		}
	})

	it("sets all platform environment variables", func() {
		root := internal.ScratchDir(t, "platform")
		internal.WriteToFile(strings.NewReader("test-value-1"), filepath.Join(root, "env", "TEST_KEY_1"), 0644)
		internal.WriteToFile(strings.NewReader("test-value-2"), filepath.Join(root, "env", "TEST_KEY_2"), 0644)

		defer internal.ProtectEnv(t, "TEST_KEY_1", "TEST_KEY_2")()

		platform, err := libbuildpack.NewPlatform(root, logger)
		if err != nil {
			t.Fatal(err)
		}

		platform.Envs.SetAll()

		if os.Getenv("TEST_KEY_1") != "test-value-1" {
			t.Errorf("os.GetEnv(\"TEST_KEY_1\") = %s, expected test-value-1", os.Getenv("TEST_KEY_1"))
		}

		if os.Getenv("TEST_KEY_2") != "test-value-2" {
			t.Errorf("os.GetEnv(\"TEST_KEY_2\") = %s, expected test-value-2", os.Getenv("TEST_KEY_2"))
		}
	})

	it("sets a platform environment variable", func() {
		root := internal.ScratchDir(t, "platform")
		internal.WriteToFile(strings.NewReader("test-value"), filepath.Join(root, "env", "TEST_KEY"), 0644)

		defer internal.ProtectEnv(t, "TEST_KEY")()

		platform, err := libbuildpack.NewPlatform(root, logger)
		if err != nil {
			t.Fatal(err)
		}

		for _, e := range platform.Envs {
			if e.Name == "TEST_KEY" {
				if err := e.Set() ; err != nil {
					t.Fatal(e)
				}
			}
		}

		if os.Getenv("TEST_KEY") != "test-value" {
			t.Errorf("os.GetEnv(\"TEST_KEY\") = %s, expected test-value", os.Getenv("TEST_KEY"))
		}
	})
}

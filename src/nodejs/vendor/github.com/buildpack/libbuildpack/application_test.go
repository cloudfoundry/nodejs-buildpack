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
	"testing"

	"github.com/buildpack/libbuildpack"
	"github.com/buildpack/libbuildpack/internal"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestApplication(t *testing.T) {
	spec.Run(t, "Application", testApplication, spec.Report(report.Terminal{}))
}

func testApplication(t *testing.T, when spec.G, it spec.S) {

	var logger = libbuildpack.NewLogger(nil, nil)

	it("returns the root of the application", func() {
		root := internal.ScratchDir(t, "application")
		application := libbuildpack.NewApplication(root, logger)

		if application.Root != root {
			t.Errorf("Application.Root = %s, wanted %s", application.Root, root)
		}
	})

	it("extracts root from working directory", func() {
		root := internal.ScratchDir(t, "application")
		defer internal.ReplaceWorkingDirectory(t, root)()

		application, err := libbuildpack.DefaultApplication(logger)
		if err != nil {
			t.Fatal(err)
		}

		if application.Root != root {
			t.Errorf("Application.Root = %s, wanted %s", application.Root, root)
		}
	})
}

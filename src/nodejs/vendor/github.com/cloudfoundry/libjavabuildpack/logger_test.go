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

package libjavabuildpack_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/buildpack/libbuildpack"
	"github.com/cloudfoundry/libjavabuildpack"
	"github.com/fatih/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestLogger(t *testing.T) {
	spec.Run(t, "Logger", testLogger, spec.Report(report.Terminal{}))
}

func testLogger(t *testing.T, when spec.G, it spec.S) {

	it("writes eye catcher on first line", func() {
		var info bytes.Buffer

		logger := libjavabuildpack.Logger{Logger: libbuildpack.NewLogger(nil, &info)}
		logger.FirstLine("test %s", "message")

		expected := fmt.Sprintf("%s test message\n", color.New(color.FgRed, color.Bold).Sprint("----->"))

		if info.String() != expected {
			t.Errorf("FirstLine = %s, expected %s", info.String(), expected)
		}
	})

	it("writes indent on second line", func() {
		var info bytes.Buffer

		logger := libjavabuildpack.Logger{Logger: libbuildpack.NewLogger(nil, &info)}
		logger.SubsequentLine("test %s", "message")

		if info.String() != "       test message\n" {
			t.Errorf("SubsequentLine = %s, expected -----> test message", info.String())
		}
	})

	it("formats pretty version for buildpack", func() {
		logger := libjavabuildpack.Logger{Logger: libbuildpack.NewLogger(nil, nil)}

		buildpack := libjavabuildpack.Buildpack{
			Buildpack: libbuildpack.Buildpack{
				Info: libbuildpack.BuildpackInfo{Name: "test-name", Version: "test-version"},
			},
		}

		actual := logger.PrettyVersion(buildpack)
		expected := fmt.Sprintf("%s %s", color.New(color.FgBlue, color.Bold).Sprint("test-name"),
			color.BlueString("test-version"))

		if actual != expected {
			t.Errorf("PrettyVersion = %s, expected %s", actual, expected)
		}
	})

	it("formats pretty version for dependency", func() {
		logger := libjavabuildpack.Logger{Logger: libbuildpack.NewLogger(nil, nil)}

		v, err := semver.NewVersion("1.0")
		if err != nil {
			t.Fatal(err)
		}

		actual := logger.PrettyVersion(libjavabuildpack.Dependency{Name: "test-name", Version: libjavabuildpack.Version{v}})
		expected := fmt.Sprintf("%s %s", color.New(color.FgBlue, color.Bold).Sprint("test-name"),
			color.BlueString("1.0"))

		if actual != expected {
			t.Errorf("PrettyVersion = %s, expected %s", actual, expected)
		}
	})
}

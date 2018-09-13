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
	"bytes"
	"io"
	"testing"

	"github.com/buildpack/libbuildpack"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestLogger(t *testing.T) {
	spec.Run(t, "Logger", testLogger, spec.Report(report.Terminal{}))
}

func testLogger(t *testing.T, when spec.G, it spec.S) {

	it("writes output to debug writer", func() {
		var debug bytes.Buffer

		logger := libbuildpack.NewLogger(&debug, nil)
		logger.Debug("%s %s", "test-string-1", "test-string-2")

		if debug.String() != "test-string-1 test-string-2\n" {
			t.Errorf("debug = %s, wanted test-string-1 test-string-2\\n", debug.String())
		}
	})

	it("does not write to debug if not configured", func() {
		var debug io.Writer

		logger := libbuildpack.NewLogger(debug, nil)
		logger.Debug("%s %s", "test-string-1", "test-string-2")
	})

	it("reports debug enabled when configured", func() {
		var debug bytes.Buffer

		if !libbuildpack.NewLogger(&debug, nil).IsDebugEnabled() {
			t.Errorf("IsDebugEnabled = false, expected true")
		}
	})

	it("reports debug disabled when not configured", func() {
		var debug io.Writer

		if libbuildpack.NewLogger(debug, nil).IsDebugEnabled() {
			t.Errorf("IsDebugEnabled = true, expected false")
		}
	})

	it("writes output to info writer", func() {
		var info bytes.Buffer

		logger := libbuildpack.NewLogger(nil, &info)
		logger.Info("%s %s", "test-string-1", "test-string-2")

		if info.String() != "test-string-1 test-string-2\n" {
			t.Errorf("info = %s, wanted test-string-1 test-string-2\\n", info.String())
		}
	})

	it("does not write to info if not configured", func() {
		var info io.Writer

		logger := libbuildpack.NewLogger(nil, info)
		logger.Info("%s %s", "test-string-1", "test-string-2")
	})

	it("reports info enabled when configured", func() {
		var info bytes.Buffer

		if !libbuildpack.NewLogger(nil, &info).IsInfoEnabled() {
			t.Errorf("IsInfoEnabled = false, expected true")
		}
	})

	it("reports info disabled when not configured", func() {
		var info io.Writer

		if libbuildpack.NewLogger(nil, info).IsInfoEnabled() {
			t.Errorf("IsInfoEnabled = true, expected false")
		}
	})
}

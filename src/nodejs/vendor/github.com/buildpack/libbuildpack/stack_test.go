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
	"testing"

	"github.com/buildpack/libbuildpack"
	"github.com/buildpack/libbuildpack/internal"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestStack(t *testing.T) {
	spec.Run(t, "Stack", testStack, spec.Report(report.Terminal{}))
}

func testStack(t *testing.T, when spec.G, it spec.S) {

	logger := libbuildpack.Logger{}

	it("extracts value from PACK_STACK_ID", func() {
		defer internal.ReplaceEnv(t, "PACK_STACK_ID", "test-stack-name")()

		actual, err := libbuildpack.DefaultStack(logger)
		if err != nil {
			t.Fatal(err)
		}

		if actual != "test-stack-name" {
			t.Errorf("DefaultStack = %s, expected test-stack-name", actual)
		}
	})

	it("returns error when PACK_STACK_ID not set", func() {
		defer internal.ProtectEnv(t, "PACK_STACK_ID")()

		os.Unsetenv("PACK_STACK_ID")

		_, err := libbuildpack.DefaultStack(logger)
		if err.Error() != "PACK_STACK_ID not set" {
			t.Errorf("DefaultStack = %s, expected PACK_STACK_ID not set", err.Error())
		}
	})
}

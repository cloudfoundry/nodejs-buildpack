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
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/libjavabuildpack"
	"github.com/cloudfoundry/libjavabuildpack/test"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestExtractTarGz(t *testing.T) {
	spec.Run(t, "ExtractTarGz", testExtractTarGz, spec.Report(report.Terminal{}))
}

func testExtractTarGz(t *testing.T, when spec.G, it spec.S) {

	when("ExtractTarGz", func() {

		it("extracts the archive", func() {
			root := test.ScratchDir(t, "util")

			err := libjavabuildpack.ExtractTarGz(test.FixturePath(t, "test-archive.tar.gz"), root, 0)
			if err != nil {
				t.Fatal(err)
			}

			test.BeFileLike(t, filepath.Join(root, "fileA.txt"), 0644, "")
			test.BeFileLike(t, filepath.Join(root, "dirA", "fileB.txt"), 0644, "")
			test.BeFileLike(t, filepath.Join(root, "dirA", "fileC.txt"), 0644, "")
		})

		it("skips stripped components", func() {
			root := test.ScratchDir(t, "util")

			err := libjavabuildpack.ExtractTarGz(test.FixturePath(t, "test-archive.tar.gz"), root, 1)
			if err != nil {
				t.Fatal(err)
			}

			exists, err := libjavabuildpack.FileExists(filepath.Join(root, "fileA.txt"))
			if err != nil {
				t.Fatal(err)
			}

			if exists {
				t.Errorf("fileA.txt exists, expected not to")
			}

			test.BeFileLike(t, filepath.Join(root, "fileB.txt"), 0644, "")
			test.BeFileLike(t, filepath.Join(root, "fileC.txt"), 0644, "")
		})

	})

	when("ExtractZip", func() {

		it("extracts the archive", func() {
			root := test.ScratchDir(t, "util")

			err := libjavabuildpack.ExtractZip(test.FixturePath(t, "test-archive.zip"), root, 0)
			if err != nil {
				t.Fatal(err)
			}

			test.BeFileLike(t, filepath.Join(root, "fileA.txt"), 0644, "")
			test.BeFileLike(t, filepath.Join(root, "dirA", "fileB.txt"), 0644, "")
			test.BeFileLike(t, filepath.Join(root, "dirA", "fileC.txt"), 0644, "")
		})

		it("skips stripped components", func() {
			root := test.ScratchDir(t, "util")

			err := libjavabuildpack.ExtractZip(test.FixturePath(t, "test-archive.zip"), root, 1)
			if err != nil {
				t.Fatal(err)
			}

			exists, err := libjavabuildpack.FileExists(filepath.Join(root, "fileA.txt"))
			if err != nil {
				t.Fatal(err)
			}

			if exists {
				t.Errorf("fileA.txt exists, expected not to")
			}

			test.BeFileLike(t, filepath.Join(root, "fileB.txt"), 0644, "")
			test.BeFileLike(t, filepath.Join(root, "fileC.txt"), 0644, "")
		})

	})

}

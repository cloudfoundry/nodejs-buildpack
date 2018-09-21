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
	"reflect"
	"strings"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/buildpack/libbuildpack"
	"github.com/cloudfoundry/libjavabuildpack"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestBuildpack(t *testing.T) {
	spec.Run(t, "Buildpack", testBuildpack, spec.Report(report.Terminal{}))
}

func testBuildpack(t *testing.T, when spec.G, it spec.S) {

	it("returns error with no defined dependencies", func() {
		b := libbuildpack.Buildpack{}

		actual, err := libjavabuildpack.Buildpack{Buildpack: b}.Dependencies()
		if err != nil {
			t.Fatal(err)
		}

		expected := libjavabuildpack.Dependencies{}
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("Buildpack.Dependencies = %s, expected %s", actual, expected)
		}
	})

	it("returns error with incorrectly defined dependencies", func() {
		b := libbuildpack.Buildpack{
			Metadata: libbuildpack.BuildpackMetadata{"dependencies": "test-dependency"},
		}

		_, err := libjavabuildpack.Buildpack{Buildpack: b}.Dependencies()

		if err.Error() != "dependencies have invalid structure" {
			t.Errorf("Buildpack.Dependencies = %s, expected dependencies have invalid structure", err.Error())
		}
	})

	it("returns dependencies", func() {
		b := libbuildpack.Buildpack{
			Metadata: libbuildpack.BuildpackMetadata{
				"dependencies": []map[string]interface{}{
					{
						"id":      "test-id-1",
						"name":    "test-name-1",
						"version": "1.0",
						"uri":     "test-uri-1",
						"sha256":  "test-sha256-1",
						"stacks":  []interface{}{"test-stack-1a", "test-stack-1b"},
					},
					{
						"id":      "test-id-2",
						"name":    "test-name-2",
						"version": "2.0",
						"uri":     "test-uri-2",
						"sha256":  "test-sha256-2",
						"stacks":  []interface{}{"test-stack-2a", "test-stack-2b"},
					},
				},
			},
		}

		expected := libjavabuildpack.Dependencies{
			libjavabuildpack.Dependency{
				ID:      "test-id-1",
				Name:    "test-name-1",
				Version: newVersion(t, "1.0"),
				URI:     "test-uri-1",
				SHA256:  "test-sha256-1",
				Stacks:  []string{"test-stack-1a", "test-stack-1b"}},
			libjavabuildpack.Dependency{
				ID:      "test-id-2",
				Name:    "test-name-2",
				Version: newVersion(t, "2.0"),
				URI:     "test-uri-2",
				SHA256:  "test-sha256-2",
				Stacks:  []string{"test-stack-2a", "test-stack-2b"}},
		}

		actual, err := libjavabuildpack.Buildpack{Buildpack: b}.Dependencies()
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("Buildpack.Dependencies = %s, expected %s", actual, expected)
		}
	})

	it("returns include_files if it exists", func() {
		b := libbuildpack.Buildpack{
			Metadata: libbuildpack.BuildpackMetadata{
				"include_files": []interface{}{"test-file-1", "test-file-2"},
			},
		}

		actual, err := libjavabuildpack.Buildpack{Buildpack: b}.IncludeFiles()
		if err != nil {
			t.Fatal(err)
		}

		expected := []string{"test-file-1", "test-file-2"}
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("Buildpack.IncludeFiles = %s, expected empty []string", actual)
		}
	})

	it("returns empty []string if include_files does not exist", func() {
		b := libbuildpack.Buildpack{}

		actual, err := libjavabuildpack.Buildpack{Buildpack: b}.IncludeFiles()
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(actual, []string{}) {
			t.Errorf("Buildpack.IncludeFiles = %s, expected empty []string", actual)
		}
	})

	it("returns false if include_files is not []string", func() {
		b := libbuildpack.Buildpack{
			Metadata: libbuildpack.BuildpackMetadata{
				"include_files": 1,
			},
		}

		_, err := libjavabuildpack.Buildpack{Buildpack: b}.IncludeFiles()
		if err.Error() != "include_files is not an array of strings" {
			t.Errorf("Buildpack.IncludeFiles = %s, expected include_files is not an array of strings", err.Error())
		}
	})

	it("returns pre_package if it exists", func() {
		b := libbuildpack.Buildpack{
			Metadata: libbuildpack.BuildpackMetadata{
				"pre_package": "test-package",
			},
		}

		actual, ok := libjavabuildpack.Buildpack{Buildpack: b}.PrePackage()
		if !ok {
			t.Errorf("Buildpack.PrePackage() = %t, expected true", ok)
		}

		if actual != "test-package" {
			t.Errorf("Buildpack.PrePackage() %s, expected test-package", actual)
		}
	})

	it("returns false if pre_package does not exist", func() {
		b := libbuildpack.Buildpack{}

		_, ok := libjavabuildpack.Buildpack{Buildpack: b}.PrePackage()
		if ok {
			t.Errorf("Buildpack.PrePackage() = %t, expected false", ok)
		}
	})

	it("returns false if pre_package is not string", func() {
		b := libbuildpack.Buildpack{
			Metadata: libbuildpack.BuildpackMetadata{
				"pre_package": 1,
			},
		}

		_, ok := libjavabuildpack.Buildpack{Buildpack: b}.PrePackage()
		if ok {
			t.Errorf("Buildpack.PrePackage() = %t, expected false", ok)
		}
	})

	it("filters by id", func() {
		d := libjavabuildpack.Dependencies{
			libjavabuildpack.Dependency{
				ID:      "test-id-1",
				Name:    "test-name",
				Version: newVersion(t, "1.0"),
				URI:     "test-uri",
				SHA256:  "test-sha256",
				Stacks:  []string{"test-stack-1", "test-stack-2"}},
			libjavabuildpack.Dependency{
				ID:      "test-id-2",
				Name:    "test-name",
				Version: newVersion(t, "1.0"),
				URI:     "test-uri",
				SHA256:  "test-sha256",
				Stacks:  []string{"test-stack-1", "test-stack-2"}},
		}

		expected := libjavabuildpack.Dependency{
			ID:      "test-id-2",
			Name:    "test-name",
			Version: newVersion(t, "1.0"),
			URI:     "test-uri",
			SHA256:  "test-sha256",
			Stacks:  []string{"test-stack-1", "test-stack-2"}}

		actual, err := d.Best("test-id-2", "1.0", "test-stack-1")
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("Dependencies.Best = %s, expected %s", actual, expected)
		}
	})

	it("filters by version constraint", func() {
		d := libjavabuildpack.Dependencies{
			libjavabuildpack.Dependency{
				ID:      "test-id",
				Name:    "test-name",
				Version: newVersion(t, "1.0"),
				URI:     "test-uri",
				SHA256:  "test-sha256",
				Stacks:  []string{"test-stack-1", "test-stack-2"}},
			libjavabuildpack.Dependency{
				ID:      "test-id",
				Name:    "test-name",
				Version: newVersion(t, "2.0"),
				URI:     "test-uri",
				SHA256:  "test-sha256",
				Stacks:  []string{"test-stack-1", "test-stack-2"}},
		}

		expected := libjavabuildpack.Dependency{
			ID:      "test-id",
			Name:    "test-name",
			Version: newVersion(t, "2.0"),
			URI:     "test-uri",
			SHA256:  "test-sha256",
			Stacks:  []string{"test-stack-1", "test-stack-2"}}

		actual, err := d.Best("test-id", "2.0", "test-stack-1")
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("Dependencies.Best = %s, expected %s", actual, expected)
		}
	})

	it("filters by stack", func() {
		d := libjavabuildpack.Dependencies{
			libjavabuildpack.Dependency{
				ID:      "test-id",
				Name:    "test-name",
				Version: newVersion(t, "1.0"),
				URI:     "test-uri",
				SHA256:  "test-sha256",
				Stacks:  []string{"test-stack-1", "test-stack-2"}},
			libjavabuildpack.Dependency{
				ID:      "test-id",
				Name:    "test-name",
				Version: newVersion(t, "1.0"),
				URI:     "test-uri",
				SHA256:  "test-sha256",
				Stacks:  []string{"test-stack-1", "test-stack-3"}},
		}

		expected := libjavabuildpack.Dependency{
			ID:      "test-id",
			Name:    "test-name",
			Version: newVersion(t, "1.0"),
			URI:     "test-uri",
			SHA256:  "test-sha256",
			Stacks:  []string{"test-stack-1", "test-stack-3"}}

		actual, err := d.Best("test-id", "1.0", "test-stack-3")
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("Dependencies.Best = %s, expected %s", actual, expected)
		}
	})

	it("returns the best dependency", func() {
		d := libjavabuildpack.Dependencies{
			libjavabuildpack.Dependency{
				ID:      "test-id",
				Name:    "test-name",
				Version: newVersion(t, "1.1"),
				URI:     "test-uri",
				SHA256:  "test-sha256",
				Stacks:  []string{"test-stack-1", "test-stack-2"}},
			libjavabuildpack.Dependency{
				ID:      "test-id",
				Name:    "test-name",
				Version: newVersion(t, "1.0"),
				URI:     "test-uri",
				SHA256:  "test-sha256",
				Stacks:  []string{"test-stack-1", "test-stack-3"}},
		}

		expected := libjavabuildpack.Dependency{
			ID:      "test-id",
			Name:    "test-name",
			Version: newVersion(t, "1.1"),
			URI:     "test-uri",
			SHA256:  "test-sha256",
			Stacks:  []string{"test-stack-1", "test-stack-2"}}

		actual, err := d.Best("test-id", "1.*", "test-stack-1")
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("Dependencies.Best = %s, expected %s", actual, expected)
		}
	})

	it("returns error if there are no matching dependencies", func() {
		d := libjavabuildpack.Dependencies{
			libjavabuildpack.Dependency{
				ID:      "test-id",
				Name:    "test-name",
				Version: newVersion(t, "1.0"),
				URI:     "test-uri",
				SHA256:  "test-sha256",
				Stacks:  []string{"test-stack-1", "test-stack-2"}},
			libjavabuildpack.Dependency{
				ID:      "test-id",
				Name:    "test-name",
				Version: newVersion(t, "1.0"),
				URI:     "test-uri",
				SHA256:  "test-sha256",
				Stacks:  []string{"test-stack-1", "test-stack-3"}},
		}

		_, err := d.Best("test-id-2", "1.0", "test-stack-1")
		if !strings.HasPrefix(err.Error(), "no valid dependencies") {
			t.Errorf("Dependencies.Best = %s, expected no valid dependencies...", err.Error())
		}
	})

	it("substitutes all wildcard for unspecified version constraint", func() {
		d := libjavabuildpack.Dependencies{
			libjavabuildpack.Dependency{
				ID:      "test-id",
				Name:    "test-name",
				Version: newVersion(t, "1.1"),
				URI:     "test-uri",
				SHA256:  "test-sha256",
				Stacks:  []string{"test-stack-1", "test-stack-2"}},
		}

		expected := libjavabuildpack.Dependency{
			ID:      "test-id",
			Name:    "test-name",
			Version: newVersion(t, "1.1"),
			URI:     "test-uri",
			SHA256:  "test-sha256",
			Stacks:  []string{"test-stack-1", "test-stack-2"}}

		actual, err := d.Best("test-id", "", "test-stack-1")
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("Dependencies.Best = %s, expected %s", actual, expected)
		}
	})
}

func newVersion(t *testing.T, version string) libjavabuildpack.Version {
	t.Helper()

	v, err := semver.NewVersion(version)
	if err != nil {
		t.Fatal(err)
	}

	return libjavabuildpack.Version{Version: v}
}

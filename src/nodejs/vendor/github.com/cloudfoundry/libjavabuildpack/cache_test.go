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
	"strings"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/buildpack/libbuildpack"
	"github.com/cloudfoundry/libjavabuildpack"
	"github.com/cloudfoundry/libjavabuildpack/internal"
	"github.com/cloudfoundry/libjavabuildpack/test"
	"github.com/h2non/gock"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestCache(t *testing.T) {
	spec.Run(t, "Cache", testCache, spec.Report(report.Terminal{}))
}

func testCache(t *testing.T, when spec.G, it spec.S) {

	when("DependencyCacheLayer", func() {

		it("creates a dependency cache with the dependency id", func() {
			root := test.ScratchDir(t, "cache")
			cache := libjavabuildpack.Cache{Cache: libbuildpack.Cache{Root: root}}
			dependency := libjavabuildpack.Dependency{ID: "test-id"}

			d := cache.DependencyLayer(dependency)

			expected := filepath.Join(root, "test-id")
			if d.Root != expected {
				t.Errorf("DependencyCacheLayer.Root = %s, expected %s", d.Root, expected)
			}
		})

		it("contributes a dependency", func() {
			root := test.ScratchDir(t, "cache")
			cache := libjavabuildpack.Cache{Cache: libbuildpack.Cache{Root: root}}

			v, err := semver.NewVersion("1.0")
			if err != nil {
				t.Fatal(err)
			}

			dependency := libjavabuildpack.Dependency{
				ID:      "test-id",
				Version: libjavabuildpack.Version{Version: v},
				SHA256:  "6f06dd0e26608013eff30bb1e951cda7de3fdd9e78e907470e0dd5c0ed25e273",
				URI:     "http://test.com/test-path",
			}

			defer gock.Off()

			gock.New("http://test.com").
				Get("/test-path").
				Reply(200).
				BodyString("test-payload")

			contributed := false

			err = cache.DependencyLayer(dependency).Contribute(func(artifact string, layer libjavabuildpack.DependencyCacheLayer) error {
				contributed = true
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}

			if !contributed {
				t.Errorf("Expected contribution but didn't contribute")
			}
		})

		it("does not contribute a dependency", func() {
			root := test.ScratchDir(t, "cache")
			cache := libjavabuildpack.Cache{Cache: libbuildpack.Cache{Root: root}}

			v, err := semver.NewVersion("1.0")
			if err != nil {
				t.Fatal(err)
			}

			dependency := libjavabuildpack.Dependency{
				ID:      "test-id",
				Version: libjavabuildpack.Version{Version: v},
				SHA256:  "6f06dd0e26608013eff30bb1e951cda7de3fdd9e78e907470e0dd5c0ed25e273",
				URI:     "http://test.com/test-path",
			}

			libjavabuildpack.WriteToFile(strings.NewReader(`id = "test-id"
name = ""
version = "1.0"
uri = "http://test.com/test-path"
sha256 = "6f06dd0e26608013eff30bb1e951cda7de3fdd9e78e907470e0dd5c0ed25e273"
`), filepath.Join(root, dependency.ID, "dependency.toml"), 0644)

			contributed := false

			err = cache.DependencyLayer(dependency).Contribute(func(artifact string, layer libjavabuildpack.DependencyCacheLayer) error {
				contributed = true
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}

			if contributed {
				t.Errorf("Expected non-contribution but did contribute")
			}
		})
	})

	when("DownloadCacheLayer", func() {

		it("creates a download cache with the dependency SHA256 name", func() {
			root := test.ScratchDir(t, "cache")
			cache := libjavabuildpack.Cache{Cache: libbuildpack.Cache{Root: root}}
			dependency := libjavabuildpack.Dependency{SHA256: "test-sha256"}

			d := cache.DownloadLayer(dependency)

			expected := filepath.Join(root, "test-sha256")
			if d.Root != expected {
				t.Errorf("DownloadCacheLayer.Root = %s, expected %s", d.Root, expected)
			}
		})

		it("downloads a dependency", func() {
			root := test.ScratchDir(t, "cache")
			cache := libjavabuildpack.Cache{Cache: libbuildpack.Cache{Root: root}}

			v, err := semver.NewVersion("1.0")
			if err != nil {
				t.Fatal(err)
			}

			dependency := libjavabuildpack.Dependency{
				Version: libjavabuildpack.Version{Version: v},
				SHA256:  "6f06dd0e26608013eff30bb1e951cda7de3fdd9e78e907470e0dd5c0ed25e273",
				URI:     "http://test.com/test-path",
			}

			defer gock.Off()

			gock.New("http://test.com").
				Get("/test-path").
				Reply(200).
				BodyString("test-payload")

			a, err := cache.DownloadLayer(dependency).Artifact()
			if err != nil {
				t.Fatal(err)
			}

			expected := filepath.Join(root, dependency.SHA256, "test-path")
			if a != expected {
				t.Errorf("DownloadCacheLayer.Artifact() = %s, expected %s", a, expected)
			}

			internal.BeFileLike(t, expected, 0644, "test-payload")

			expected = filepath.Join(root, dependency.SHA256, "dependency.toml")
			internal.BeFileLike(t, expected, 0644, `id = ""
name = ""
version = "1.0"
uri = "http://test.com/test-path"
sha256 = "6f06dd0e26608013eff30bb1e951cda7de3fdd9e78e907470e0dd5c0ed25e273"
`)
		})

		it("does not download a cached dependency", func() {
			root := test.ScratchDir(t, "cache")
			cache := libjavabuildpack.Cache{Cache: libbuildpack.Cache{Root: root}}

			v, err := semver.NewVersion("1.0")
			if err != nil {
				t.Fatal(err)
			}

			dependency := libjavabuildpack.Dependency{
				Version: libjavabuildpack.Version{Version: v},
				SHA256:  "6f06dd0e26608013eff30bb1e951cda7de3fdd9e78e907470e0dd5c0ed25e273",
				URI:     "http://test.com/test-path",
			}

			libjavabuildpack.WriteToFile(strings.NewReader(`id = ""
name = ""
version = "1.0"
uri = "http://test.com/test-path"
sha256 = "6f06dd0e26608013eff30bb1e951cda7de3fdd9e78e907470e0dd5c0ed25e273"
`), filepath.Join(root, dependency.SHA256, "dependency.toml"), 0644)

			_, err = cache.DownloadLayer(dependency).Artifact()
			if err != nil {
				t.Fatal(err)
			}
		})

		it("returns metadata location", func() {
			root := test.ScratchDir(t, "cache")
			cache := libjavabuildpack.Cache{Cache: libbuildpack.Cache{Root: root}}

			v, err := semver.NewVersion("1.0")
			if err != nil {
				t.Fatal(err)
			}

			dependency := libjavabuildpack.Dependency{
				Version: libjavabuildpack.Version{Version: v},
				SHA256:  "6f06dd0e26608013eff30bb1e951cda7de3fdd9e78e907470e0dd5c0ed25e273",
				URI:     "http://test.com/test-path",
			}

			libjavabuildpack.WriteToFile(strings.NewReader(`id = ""
name = ""
version = "1.0"
uri = "http://test.com/test-path"
sha256 = "6f06dd0e26608013eff30bb1e951cda7de3fdd9e78e907470e0dd5c0ed25e273"
`), filepath.Join(root, dependency.SHA256, "dependency.toml"), 0644)

			actual := cache.DownloadLayer(dependency).Metadata()

			expected := filepath.Join(root, dependency.SHA256, "dependency.toml")
			if actual != expected {
				t.Errorf("DownloadLayer.Metadata() = %s, expected %s", actual, expected)
			}
		})
	})

}

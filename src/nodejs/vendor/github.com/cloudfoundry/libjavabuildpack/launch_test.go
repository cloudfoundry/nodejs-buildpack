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
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/buildpack/libbuildpack"
	"github.com/cloudfoundry/libjavabuildpack"
	"github.com/cloudfoundry/libjavabuildpack/test"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestLaunch(t *testing.T) {
	spec.Run(t, "Launch", testLaunch, spec.Report(report.Terminal{}))
}

func testLaunch(t *testing.T, when spec.G, it spec.S) {

	it("creates a dependency launch with the dependency id", func() {
		root := test.ScratchDir(t, "cache")
		launch := libjavabuildpack.Launch{Launch: libbuildpack.Launch{Root: root}}
		dependency := libjavabuildpack.Dependency{ID: "test-id"}

		d := launch.DependencyLayer(dependency)

		expected := filepath.Join(root, "test-id")
		if d.Root != expected {
			t.Errorf("DependencyLaunchLayer.Root = %s, expected %s", d.Root, expected)
		}
	})

	it("calls contributor to contribute launch layer", func() {
		root := test.ScratchDir(t, "cache")
		cache := libjavabuildpack.Cache{Cache: libbuildpack.Cache{Root: root}}
		launch := libjavabuildpack.Launch{Launch: libbuildpack.Launch{Root: root}, Cache: cache}

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
`), filepath.Join(root, dependency.SHA256, "dependency.toml"), 0644)

		contributed := false

		err = launch.DependencyLayer(dependency).Contribute(func(artifact string, layer libjavabuildpack.DependencyLaunchLayer) error {
			contributed = true;
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}

		if !contributed {
			t.Errorf("Expected contribution but didn't contribute")
		}
	})

	it("does not call contributor for a cached launch layer", func() {
		root := test.ScratchDir(t, "cache")
		cache := libjavabuildpack.Cache{Cache: libbuildpack.Cache{Root: root}}
		launch := libjavabuildpack.Launch{Launch: libbuildpack.Launch{Root: root}, Cache: cache}

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
`), filepath.Join(root, fmt.Sprintf("%s.toml", dependency.ID)), 0644)

		contributed := false

		err = launch.DependencyLayer(dependency).Contribute(func(artifact string, layer libjavabuildpack.DependencyLaunchLayer) error {
			contributed = true;
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}

		if contributed {
			t.Errorf("Expected non-contribution but did contribute")
		}
	})

	it("returns artifact name", func() {
		root := test.ScratchDir(t, "cache")
		launch := libjavabuildpack.Launch{Launch: libbuildpack.Launch{Root: root}}
		dependency := libjavabuildpack.Dependency{ID: "test-id", URI: "http://localhost/path/test-artifact-name"}

		d := launch.DependencyLayer(dependency)

		if d.ArtifactName() != "test-artifact-name" {
			t.Errorf("DependencyLaunchLayer.ArtifactName = %s, expected test-artifact-name", d.ArtifactName())
		}
	})

}

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

package test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/buildpack/libbuildpack"
	"github.com/cloudfoundry/libjavabuildpack"
	"github.com/cloudfoundry/libjavabuildpack/internal"
)

// BuildFactory is a factory for creating a test Build.
type BuildFactory struct {
	Build libjavabuildpack.Build
}

// AddBuildPlan adds an entry to a build plan.
func (f *BuildFactory) AddBuildPlan(t *testing.T, name string, dependency libbuildpack.BuildPlanDependency) {
	t.Helper()
	f.Build.BuildPlan[name] = dependency
}

// AddDependency adds a dependency to the buildpack metadata and copies a fixture into a cached dependency layer.
func (f *BuildFactory) AddDependency(t *testing.T, id string, fixture string) {
	t.Helper()

	d := f.newDependency(t, id, fixture)
	f.cacheFixture(t, d, fixture)
	f.addDependency(t, d)
}

func (f *BuildFactory) addDependency(t *testing.T, dependency libjavabuildpack.Dependency) {
	t.Helper()

	metadata := f.Build.Buildpack.Metadata
	dependencies := metadata["dependencies"].([]map[string]interface{})

	var stacks []interface{}
	for _, stack := range dependency.Stacks {
		stacks = append(stacks, stack)
	}

	metadata["dependencies"] = append(dependencies, map[string]interface{}{
		"id":      dependency.ID,
		"name":    dependency.Name,
		"version": dependency.Version.Version.Original(),
		"uri":     dependency.URI,
		"sha256":  dependency.SHA256,
		"stacks":  stacks,
	})
}

func (f *BuildFactory) cacheFixture(t *testing.T, dependency libjavabuildpack.Dependency, fixture string) {
	t.Helper()

	l := f.Build.Cache.Layer(dependency.SHA256)
	if err := libjavabuildpack.CopyFile(filepath.Join(FindRoot(t), "fixtures", fixture), filepath.Join(l.Root, filepath.Base(fixture))); err != nil {
		t.Fatal(err)
	}

	d, err := internal.ToTomlString(dependency)
	if err != nil {
		t.Fatal(err)
	}
	if err := libjavabuildpack.WriteToFile(strings.NewReader(d), filepath.Join(l.Root, "dependency.toml"), 0644); err != nil {
		t.Fatal(err)
	}
}

func (f *BuildFactory) newDependency(t *testing.T, id string, fixture string) libjavabuildpack.Dependency {
	t.Helper()

	version, err := semver.NewVersion("1.0")
	if err != nil {
		t.Fatal(err)
	}

	return libjavabuildpack.Dependency{
		ID:      id,
		Name:    "test-name",
		Version: libjavabuildpack.Version{Version: version},
		URI:     fmt.Sprintf("http://localhost/%s", filepath.Base(fixture)),
		SHA256:  "test-hash",
		Stacks:  libjavabuildpack.Stacks{f.Build.Stack},
	}
}

// NewBuildFactory creates a new instance of BuildFactory.
func NewBuildFactory(t *testing.T) BuildFactory {
	t.Helper()
	f := BuildFactory{}

	root := ScratchDir(t, "test-build-factory")

	f.Build.Application.Root = filepath.Join(root, "app")
	f.Build.BuildPlan = make(libbuildpack.BuildPlan)

	f.Build.Buildpack.Metadata = make(libbuildpack.BuildpackMetadata)
	f.Build.Buildpack.Metadata["dependencies"] = make([]map[string]interface{}, 0)

	f.Build.Cache.Root = filepath.Join(root, "cache")

	f.Build.Launch.Root = filepath.Join(root, "launch")
	f.Build.Launch.Cache = f.Build.Cache

	f.Build.Platform.Root = filepath.Join(root, "platform")

	return f
}

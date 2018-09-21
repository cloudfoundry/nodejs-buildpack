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

package libjavabuildpack

import (
	"fmt"
	"sort"

	"github.com/Masterminds/semver"
	"github.com/buildpack/libbuildpack"
)

// Buildpack is an extension to libbuildpack.Buildpack that adds additional opinionated behaviors.
type Buildpack struct {
	libbuildpack.Buildpack
}

// Dependencies returns the collection of dependencies extracted from the generic buildpack metadata.
func (b Buildpack) Dependencies() (Dependencies, error) {
	d, ok := b.Metadata["dependencies"]
	if !ok {
		return Dependencies{}, nil
	}

	deps, ok := d.([]map[string]interface{})
	if !ok {
		return Dependencies{}, fmt.Errorf("dependencies have invalid structure")
	}

	var dependencies Dependencies
	for _, dep := range deps {
		d, err := b.dependency(dep)
		if err != nil {
			return Dependencies{}, err
		}

		dependencies = append(dependencies, d)
	}

	b.Logger.Debug("Dependencies: %s", dependencies)
	return dependencies, nil
}

// IncludeFiles returns the include_files buildpack metadata.
func (b Buildpack) IncludeFiles() ([]string, error) {
	i, ok := b.Metadata["include_files"]
	if !ok {
		return []string{}, nil
	}

	files, ok := i.([]interface{})
	if !ok {
		return []string{}, fmt.Errorf("include_files is not an array of strings")
	}

	var includes []string
	for _, candidate := range files {
		file, ok := candidate.(string)
		if !ok {
			return []string{}, fmt.Errorf("include_files is not an array of strings")
		}

		includes = append(includes, file)
	}

	return includes, nil
}

// PrePackage returns the pre_package buildpack metadata.
func (b Buildpack) PrePackage() (string, bool) {
	p, ok := b.Metadata["pre_package"]
	if !ok {
		return "", false
	}

	s, ok := p.(string)
	return s, ok
}

func (b Buildpack) dependency(dep map[string]interface{}) (Dependency, error) {
	id, ok := dep["id"].(string)
	if !ok {
		return Dependency{}, fmt.Errorf("dependency id missing or wrong format")
	}

	name, ok := dep["name"].(string)
	if !ok {
		return Dependency{}, fmt.Errorf("dependency name missing or wrong format")
	}

	v, ok := dep["version"].(string)
	if !ok {
		return Dependency{}, fmt.Errorf("dependency version missing or wrong format")
	}

	version, err := semver.NewVersion(v)
	if err != nil {
		return Dependency{}, err
	}

	uri, ok := dep["uri"].(string)
	if !ok {
		return Dependency{}, fmt.Errorf("dependency uri missing or wrong format")
	}

	sha256, ok := dep["sha256"].(string)
	if !ok {
		return Dependency{}, fmt.Errorf("dependency sha256 missing or wrong format")
	}

	s, ok := dep["stacks"].([]interface{})
	if !ok {
		return Dependency{}, fmt.Errorf("dependency stacks missing or wrong format")
	}

	var stacks []string
	for _, t := range s {
		u, ok := t.(string)
		if !ok {
			return Dependency{}, fmt.Errorf("dependency stacks missing or wrong format")
		}

		stacks = append(stacks, u)
	}

	return Dependency{id, name, Version{version}, uri, sha256, stacks}, nil
}

// Dependencies is a collection of Dependency instances.
type Dependencies []Dependency

// Best returns the best (latest version) dependency within a collection of Dependencies.  The candidate set is first
// filtered by id, version, and stack, then the remaining candidates are sorted for the best result.  If the
// versionConstraint is not specified (""), then the latest wildcard ("*") is used.
func (d Dependencies) Best(id string, versionConstraint string, stack string) (Dependency, error) {
	var candidates Dependencies

	vc := versionConstraint
	if vc == "" {
		vc = "*"
	}

	constraint, err := semver.NewConstraint(vc)
	if err != nil {
		return Dependency{}, err
	}

	for _, c := range d {
		if c.ID == id && constraint.Check(c.Version.Version) && c.Stacks.contains(stack) {
			candidates = append(candidates, c)
		}
	}

	if len(candidates) == 0 {
		return Dependency{}, fmt.Errorf("no valid dependencies for %s, %s, and %s in %s", id, versionConstraint, stack, d)
	}

	sort.Sort(candidates)

	return candidates[len(candidates)-1], nil
}

// Len makes Dependencies satisfy the sort.Interface interface.
func (d Dependencies) Len() int {
	return len(d)
}

// Less makes Dependencies satisfy the sort.Interface interface.
func (d Dependencies) Less(i int, j int) bool {
	return d[i].Version.LessThan(d[j].Version.Version)
}

// Swap makes Dependencies satisfy the sort.Interface interface.
func (d Dependencies) Swap(i int, j int) {
	d[i], d[j] = d[j], d[i]
}

// Dependency represents a buildpack dependency.
type Dependency struct {
	// ID is the dependency ID.
	ID string `toml:"id"`

	// Name is the dependency ID.
	Name string `toml:"name"`

	// Version is the dependency version.
	Version Version `toml:"version"`

	// URI is the dependency URI.
	URI string `toml:"uri"`

	// SHA256 is the hash of the dependency.
	SHA256 string `toml:"sha256"`

	// Stacks are the stacks the dependency is compatible with.
	Stacks Stacks `toml:"stacks"`
}

// String makes Dependency satisfy the Stringer interface.
func (d Dependency) String() string {
	return fmt.Sprintf("Dependency{ ID: %s, Name: %s, Version: %s, URI: %s, SHA256: %s, Stacks: %s}",
		d.ID, d.Name, d.Version, d.URI, d.SHA256, d.Stacks)
}

type Version struct {
	*semver.Version
}

// MarshalText makes Version satisfy the encoding.TextMarshaler interface.
func (v Version) MarshalText() ([]byte, error) {
	return []byte(v.Version.Original()), nil
}

// UnmarshalText makes Version satisfy the encoding.TextUnmarshaler interface.
func (v *Version) UnmarshalText(text []byte) error {
	s := string(text)

	w, err := semver.NewVersion(s)
	if err != nil {
		return fmt.Errorf("invalid semantic version %s", s)
	}

	v.Version = w
	return nil
}

// Stacks is a collection of stack ids
type Stacks []string

func (s Stacks) contains(stack string) bool {
	for _, v := range s {
		if v == stack {
			return true
		}
	}

	return false
}

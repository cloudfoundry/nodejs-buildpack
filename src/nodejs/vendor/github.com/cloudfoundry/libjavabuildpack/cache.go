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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/buildpack/libbuildpack"
	"github.com/cloudfoundry/libjavabuildpack/internal"
	"github.com/fatih/color"
)

// Cache is an extension to libbuildpack.Cache that allows additional functionality to be added.
type Cache struct {
	libbuildpack.Cache

	// Logger is used to write debug and info to the console.
	Logger Logger
}

// DependencyLayer returns a DependencyCacheLayer unique to a dependency.
func (c Cache) DependencyLayer(dependency Dependency) DependencyCacheLayer {
	return DependencyCacheLayer{
		c.Layer(dependency.ID),
		c.Logger,
		dependency,
		c.DownloadLayer(dependency),
	}
}

// DownloadLayer returns a DownloadCacheLayer unique to a dependency.
func (c Cache) DownloadLayer(dependency Dependency) DownloadCacheLayer {
	return DownloadCacheLayer{
		c.Layer(dependency.SHA256),
		c.Logger,
		dependency,
	}
}

// String makes Cache satisfy the Stringer interface.
func (c Cache) String() string {
	return fmt.Sprintf("Cache{ Cache: %s, Logger: %s }", c.Cache, c.Logger)
}

// DependencyCacheLayer is an extension to CacheLayer that is unique to a dependency contribution.
type DependencyCacheLayer struct {
	libbuildpack.CacheLayer

	// Logger is used to write debug and info to the console.
	Logger Logger

	dependency    Dependency
	downloadLayer DownloadCacheLayer
}

// CacheContributor defines a callback function that is called when a dependency needs to be contributed.
type CacheContributor func(artifact string, layer DependencyCacheLayer) error

// Contribute contributes an artifact to a cache layer.  If the artifact has already been contributed, the cache will be
// validated and used directly.
func (d DependencyCacheLayer) Contribute(contributor CacheContributor) error {
	m, err := d.readMetadata()
	if err != nil {
		return err
	}

	if reflect.DeepEqual(d.dependency, m) {
		d.Logger.FirstLine("%s: %s cached dependency",
			d.Logger.PrettyVersion(d.dependency), color.GreenString("Reusing"))
		return nil
	}

	d.Logger.Debug("Dependency metadata %s does not match expected %s", m, d.dependency)

	d.Logger.FirstLine("%s: %s to cache",
		d.Logger.PrettyVersion(d.dependency), color.YellowString("Contributing"))

	if err := os.RemoveAll(d.Root); err != nil {
		return err
	}

	a, err := d.downloadLayer.Artifact()
	if err != nil {
		return err
	}

	if err := contributor(a, d); err != nil {
		d.Logger.Debug("Error during contribution")
		return err
	}

	return d.writeMetadata()
}

func (d DependencyCacheLayer) metadataPath() string {
	return filepath.Join(d.Root, "dependency.toml")
}

func (d DependencyCacheLayer) readMetadata() (Dependency, error) {
	f := d.metadataPath()

	exists, err := FileExists(f)
	if err != nil || !exists {
		d.Logger.Debug("Dependency metadata %s does not exist", f)
		return Dependency{}, err
	}

	var dep Dependency

	if err = FromTomlFile(f, &dep); err != nil {
		d.Logger.Debug("Dependency metadata %s is not structured correctly", f)
		return Dependency{}, err
	}

	d.Logger.Debug("Reading dependency metadata: %s => %s", f, dep)
	return dep, nil
}

func (d DependencyCacheLayer) writeMetadata() error {
	f := d.metadataPath()
	d.Logger.Debug("Writing dependency metadata: %s <= %s", f, d.dependency)

	toml, err := internal.ToTomlString(d.dependency)
	if err != nil {
		return err
	}

	return WriteToFile(strings.NewReader(toml), f, 0644)
}

// String makes DependencyCacheLayer satisfy the Stringer interface.
func (d DependencyCacheLayer) String() string {
	return fmt.Sprintf("DependencyCacheLayer{ CacheLayer: %s, Logger: %s, dependency: %s }",
		d.CacheLayer, d.Logger, d.dependency)
}

// DownloadCacheLayer is an extension to CacheLayer that is unique to a dependency download.
type DownloadCacheLayer struct {
	libbuildpack.CacheLayer

	// Logger is used to write debug and info to the console.
	Logger Logger

	dependency Dependency
}

// Artifact returns the path to an artifact cached in the layer.  If the artifact has already been downloaded, the cache
// will be validated and used directly.
func (d DownloadCacheLayer) Artifact() (string, error) {
	a := filepath.Join(d.Root, filepath.Base(d.dependency.URI))

	m, err := d.readMetadata()
	if err != nil {
		return "", err
	}

	if reflect.DeepEqual(d.dependency, m) {
		d.Logger.SubsequentLine("%s cached download", color.GreenString("Reusing"))
		return a, nil
	}

	d.Logger.Debug("Download metadata %s does not match expected %s", m, d.dependency)

	d.Logger.SubsequentLine("%s from %s", color.YellowString("Downloading"), d.dependency.URI)

	err = d.download(a)
	if err != nil {
		return "", err
	}

	d.Logger.SubsequentLine("Verifying checksum")
	err = d.verify(a)
	if err != nil {
		return "", err
	}

	if err := d.writeMetadata(); err != nil {
		return "", err
	}

	return a, nil
}

// Metadata returns the path to the metadata file for an artifact cached in the later.
func (d DownloadCacheLayer) Metadata() string {
	return filepath.Join(d.Root, "dependency.toml")
}

// String makes DownloadCacheLayer satisfy the Stringer interface.
func (d DownloadCacheLayer) String() string {
	return fmt.Sprintf("DownloadCacheLayer{ CacheLayer: %s, Logger: %s, dependency: %s }",
		d.CacheLayer, d.Logger, d.dependency)
}

func (d DownloadCacheLayer) download(file string) error {
	resp, err := http.Get(d.dependency.URI)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("could not download: %bd", resp.StatusCode)
	}

	return WriteToFile(resp.Body, file, 0644)
}

func (d DownloadCacheLayer) readMetadata() (Dependency, error) {
	f := d.Metadata()

	exists, err := FileExists(f)
	if err != nil || !exists {
		d.Logger.Debug("Download metadata %s does not exist", f)
		return Dependency{}, err
	}

	var dep Dependency

	if err = FromTomlFile(f, &dep); err != nil {
		d.Logger.Debug("Download metadata %s is not structured correctly", f)
		return Dependency{}, err
	}

	d.Logger.Debug("Reading download metadata: %s => %s", f, dep)
	return dep, nil
}

func (d DownloadCacheLayer) verify(file string) error {
	s := sha256.New()

	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(s, f)
	if err != nil {
		return err
	}

	actualSha256 := hex.EncodeToString(s.Sum(nil))

	if actualSha256 != d.dependency.SHA256 {
		return fmt.Errorf("dependency sha256 mismatch: expected sha256 %s, actual sha256 %s",
			d.dependency.SHA256, actualSha256)
	}
	return nil
}

func (d DownloadCacheLayer) writeMetadata() error {
	f := d.Metadata()
	d.Logger.Debug("Writing cache metadata: %s <= %s", f, d.dependency)

	toml, err := internal.ToTomlString(d.dependency)
	if err != nil {
		return err
	}

	return WriteToFile(strings.NewReader(toml), f, 0644)
}

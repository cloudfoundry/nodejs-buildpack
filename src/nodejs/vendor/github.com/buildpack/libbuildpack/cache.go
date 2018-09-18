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

package libbuildpack

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/buildpack/libbuildpack/internal"
)

// Cache represents cache layers for an application.
type Cache struct {
	// Root is the path to the root directory for the caches.
	Root string

	// Logger is used to write debug and info to the console.
	Logger Logger
}

// Layer creates a CacheLayer with a specified name.
func (c Cache) Layer(name string) CacheLayer {
	return CacheLayer{filepath.Join(c.Root, name), c.Logger}
}

// String makes Cache satisfy the Stringer interface.
func (c Cache) String() string {
	return fmt.Sprintf("Cache{ Root: %s, Logger: %s }", c.Root, c.Logger)
}

// CacheLayer represents a cache layer for an application.
type CacheLayer struct {
	// Root is the path to the root directory for the cache layer.
	Root string

	// Logger is used to write debug and info to the console.
	Logger Logger
}

// AppendEnv appends the value of this environment variable to any previous declarations of the value without any
// delimitation.  If delimitation is important during concatenation, callers are required to add it.
func (c CacheLayer) AppendEnv(name string, format string, args ...interface{}) error {
	return c.addEnvFile(fmt.Sprintf("%s.append", name), format, args...)
}

// AppendPathEnv appends the value of this environment variable to any previous declarations of the value using the OS
// path delimiter.
func (c CacheLayer) AppendPathEnv(name string, format string, args ...interface{}) error {
	return c.addEnvFile(name, format, args...)
}

// Override overrides any existing value for an environment variable with this value.
func (c CacheLayer) OverrideEnv(name string, format string, args ...interface{}) error {
	return c.addEnvFile(fmt.Sprintf("%s.override", name), format, args...)
}

// String makes CacheLayer satisfy the Stringer interface.
func (c CacheLayer) String() string {
	return fmt.Sprintf("CacheLayer{ Root: %s, Logger: %s }", c.Root, c.Logger)
}

func (c CacheLayer) addEnvFile(file string, format string, args ...interface{}) error {
	f := filepath.Join(c.Root, "env", file)
	v := fmt.Sprintf(format, args...)

	c.Logger.Debug("Writing environment variable: %s <= %s", f, v)

	return internal.WriteToFile(strings.NewReader(v), f, 0644)
}

// DefaultCache creates a new instance of Cache, extracting the Root path from os.Args[2].
func DefaultCache(logger Logger) (Cache, error) {
	root, err := internal.OsArgs(2)
	if err != nil {
		return Cache{}, err
	}

	logger.Debug("Cache root: %s", root)

	return Cache{root, logger}, nil
}

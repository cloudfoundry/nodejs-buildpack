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

	"github.com/buildpack/libbuildpack"
)

// Build is an extension to libbuildpack.Build that allows additional functionality to be added.
type Build struct {
	libbuildpack.Build

	// Buildpack represents the metadata associated with a buildpack.
	Buildpack Buildpack

	// Cache represents the cache layers contributed by a buildpack.
	Cache Cache

	// Launch represents the launch layers contributed by the buildpack.
	Launch Launch

	// Logger is used to write debug and info to the console.
	Logger Logger
}

// String makes Build satisfy the Stringer interface.
func (b Build) String() string {
	return fmt.Sprintf("Build{ Build: %s, Buildpack: %s, Cache: %s, Logger: %s }",
		b.Build, b.Buildpack, b.Cache, b.Logger)
}

// DefaultBuild creates a new instance of Build using default values.
func DefaultBuild() (Build, error) {
	b, err := libbuildpack.DefaultBuild()
	if err != nil {
		return Build{}, err
	}

	logger := Logger{b.Logger}
	cache := Cache{b.Cache, logger}

	return Build{
		b,
		Buildpack{b.Buildpack},
		cache,
		Launch{b.Launch, cache, logger},
		logger,
	}, nil
}

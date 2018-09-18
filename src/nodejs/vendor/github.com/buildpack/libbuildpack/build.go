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
	"io"
	"os"
)

// Build represents all of the components available to a buildpack at build time.
type Build struct {
	// Application is the application being processed by the buildpack.
	Application Application

	// Buildpack represents the metadata associated with a buildpack.
	Buildpack Buildpack

	// BuildPlan represents dependencies contributed by previous builds.
	BuildPlan BuildPlan

	// Cache represents the cache layers contributed by a buildpack.
	Cache Cache

	// Launch represents the launch layers contributed by a buildpack.
	Launch Launch

	// Logger is used to write debug and info to the console.
	Logger Logger

	// Platform represents components contributed by the platform to the buildpack.
	Platform Platform

	// Stack is the stack currently available to the application.
	Stack string
}

// Failure signals an unsuccessful build by exiting with a specified positive status code.  This should be the final
// function called in building.
func (b Build) Failure(code int) {
	b.Logger.Debug("Build failed. Exiting with %d.", code)
	b.Logger.Info("")
	os.Exit(code)
}

// String makes Build satisfy the Stringer interface.
func (b Build) String() string {
	return fmt.Sprintf("Build{ Application: %s, Buildpack: %s, BuildPlan: %s, Cache: %s, Launch: %s, Logger: %s, Platform: %s, Stack: %s }",
		b.Application, b.Buildpack, b.BuildPlan, b.Cache, b.Launch, b.Logger, b.Platform, b.Stack)
}

// Success signals a successful build by exiting with a zero status code.  This should be the final function called in
// building.
func (b Build) Success() {
	b.Logger.Debug("Build success. Exiting with %d.", 0)
	b.Logger.Info("")
	os.Exit(0)
}

func (b Build) defaultLogger() Logger {
	var debug io.Writer

	if _, ok := os.LookupEnv("BP_DEBUG"); ok {
		debug = os.Stderr
	}

	return NewLogger(debug, os.Stdout)
}

// DefaultBuild creates a new instance of Build using default values.
func DefaultBuild() (Build, error) {
	b := Build{}

	logger := b.defaultLogger()

	application, err := DefaultApplication(logger)
	if err != nil {
		return Build{}, err
	}

	buildpack, err := DefaultBuildpack(logger)
	if err != nil {
		return Build{}, err
	}

	buildPlan, err := DefaultBuildPlan(logger)
	if err != nil {
		return Build{}, err
	}

	cache, err := DefaultCache(logger)
	if err != nil {
		return Build{}, err
	}

	launch, err := DefaultLaunch(logger)
	if err != nil {
		return Build{}, err
	}

	platform, err := DefaultPlatform(logger)
	if err != nil {
		return Build{}, err
	}

	stack, err := DefaultStack(logger)
	if err != nil {
		return Build{}, err
	}

	b.Application = application
	b.Buildpack = buildpack
	b.BuildPlan = buildPlan
	b.Cache = cache
	b.Launch = launch
	b.Logger = logger
	b.Platform = platform
	b.Stack = stack

	return b, nil
}

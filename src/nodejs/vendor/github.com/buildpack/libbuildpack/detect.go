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

// Detect represents all of the components available to a buildpack at detect time.
type Detect struct {
	// Application is the application being processed by the buildpack.
	Application Application

	// Buildpack represents the metadata associated with a buildpack.
	Buildpack Buildpack

	// BuildPlan represents dependencies contributed by previous builds.
	BuildPlan BuildPlan

	// Logger is used to write debug and info to the console.
	Logger Logger

	// Stack is the stack currently available to the application.
	Stack Stack
}

// Error signals an error during detection by exiting with a specified non-zero, non-100 status code.  This should the
// final function called in detection.
func (d Detect) Error(code int) {
	d.Logger.Debug("Detection produced an error. Exiting with %d.", code)
	os.Exit(code)
}

// Fail signals an unsuccessful detection by exiting with a 100 status code.  This should be the final function called
// in detection.
func (d Detect) Fail() {
	d.Logger.Debug("Detection failed. Exiting with %d.", 100)
	os.Exit(100)
}

// Pass signals a successful detection by exiting with a 0 status code.  This should be the final function called in
// detection.
func (d Detect) Pass(buildPlan BuildPlan) {
	d.Logger.Debug("Detection passed. Exiting with %d.", 0)

	s, err := toTomlString(buildPlan)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(102)
	}

	fmt.Fprint(os.Stdout, s)
	os.Exit(0)
}

// String makes Detect satisfy the Stringer interface.
func (d Detect) String() string {
	return fmt.Sprintf("Detect{ Application: %s, Buildpack: %s, BuildPlan: %s, Logger: %s, Stack: %s }",
		d.Application, d.Buildpack, d.BuildPlan, d.Logger, d.Stack)
}

func (d Detect) defaultLogger() Logger {
	var debug io.Writer

	if _, ok := os.LookupEnv("BP_DEBUG"); ok {
		debug = os.Stderr
	}

	return NewLogger(debug, os.Stderr)
}

// DefaultDetect creates a new instance of Detect using default values.
func DefaultDetect() (Detect, error) {
	d := Detect{}

	logger := d.defaultLogger()

	application, err := DefaultApplication(logger)
	if err != nil {
		return Detect{}, err
	}

	buildpack, err := DefaultBuildpack(logger)
	if err != nil {
		return Detect{}, err
	}

	buildPlan, err := DefaultBuildPlan(logger)
	if err != nil {
		return Detect{}, err
	}

	stack, err := DefaultStack(logger)
	if err != nil {
		return Detect{}, err
	}

	d.Application = application
	d.Buildpack = buildpack
	d.BuildPlan = buildPlan
	d.Logger = logger
	d.Stack = stack

	return d, nil
}

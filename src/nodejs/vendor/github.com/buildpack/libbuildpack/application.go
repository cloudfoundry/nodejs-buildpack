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
	"os"

	"github.com/buildpack/libbuildpack/internal"
)

// Application represents the application being processed by buildpacks.
type Application struct {
	// Root is the path to the root directory of the application.
	Root string

	// Logger is used to write debug and info to the console.
	Logger Logger
}

// String makes Application satisfy the Stringer interface.
func (a Application) String() string {
	return fmt.Sprintf("Application{ Root: %s, Logger: %s }", a.Root, a.Logger)
}

// DefaultApplication creates a new instance of Application, extracting the Root path from the working directory.
func DefaultApplication(logger Logger) (Application, error) {
	root, err := os.Getwd()
	if err != nil {
		return Application{}, err
	}

	return NewApplication(root, logger), nil
}

// NewApplication creates a new instance of Application, configuring the Root path.
func NewApplication(root string, logger Logger) Application {
	a := Application{root, logger}

	if logger.IsDebugEnabled() {
		logger.Debug("Application contents: %s", internal.DirectoryContents(root))
	}

	return a
}

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
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/libjavabuildpack"
)

// DetectFactory is a factory for creating a test Detect.
type DetectFactory struct {
	Detect libjavabuildpack.Detect
}

// NewDetectFactory creates a new instance of DetectFactory.
func NewDetectFactory(t *testing.T) DetectFactory {
	t.Helper()

	f := DetectFactory{}

	root := ScratchDir(t, "test-detect-factory")
	f.Detect.Application.Root = filepath.Join(root, "app")

	return f
}

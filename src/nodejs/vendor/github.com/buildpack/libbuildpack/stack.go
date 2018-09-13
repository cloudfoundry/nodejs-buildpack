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
)

// Stack is the name of a stack
type Stack string

// DefaultStack creates a new instance of Stack, extracting the name from the PACK_STACK_ID environment variable.
func DefaultStack(logger Logger) (Stack, error) {
	s, ok := os.LookupEnv("PACK_STACK_ID")

	if !ok {
		return "", fmt.Errorf("PACK_STACK_ID not set")
	}

	logger.Debug("Stack: %s", s)
	return Stack(s), nil
}


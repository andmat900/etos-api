// Copyright Axis Communications AB.
//
// For a full list of individual contributors, please see the commit history.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package v1alpha1

import (
	"fmt"
	"strconv"
)

// toInt tries to convert the input (float or a string) to an integer
func toInt(v interface{}) (int, error) {
	switch v := v.(type) {
	case float64:
		return int(v), nil
	case string:
		c, err := strconv.Atoi(v)
		if err != nil {
			return 0, err
		}
		return c, nil
	default:
		return 0, fmt.Errorf("conversion to %T to int not supported", v)
	}
}

// fair calculates and returns a "fair" number of iuts to use given the provided list of Iuts and greed.
func fair(size int, greed float64) int {
	if l := int(float64(size) * greed); l > 0 {
		return l
	}
	return 1
}

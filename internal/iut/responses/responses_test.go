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

package responses

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test that RespondWithJSON writes the correct HTTP code, message and adds a content type header.
func TestRespondWithJSON(t *testing.T) {
	responseRecorder := httptest.NewRecorder()
	RespondWithJSON(responseRecorder, 200, map[string]string{"hello": "world"})
	assert.Equal(t, "application/json", responseRecorder.Header().Get("Content-Type"))
	assert.Equal(t, 200, responseRecorder.Result().StatusCode)
	assert.JSONEq(t, `{"hello": "world"}`, responseRecorder.Body.String())
}

// Test that RespondWithError writes the correct HTTP code, message and adds a content type header.
func TestRespondWithError(t *testing.T) {
	responseRecorder := httptest.NewRecorder()
	RespondWithError(responseRecorder, 400, "failure")
	assert.Equal(t, "application/json", responseRecorder.Header().Get("Content-Type"))
	assert.Equal(t, 400, responseRecorder.Result().StatusCode)
	assert.JSONEq(t, `{"error": "failure"}`, responseRecorder.Body.String())
}

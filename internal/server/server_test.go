// Copyright Axis Communications AB.
//
// For a full list of individual contributors, please see the commit history.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"net/http"
	"testing"

	"github.com/eiffel-community/etos-api/test/testconfig"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// TestNewWebserver tests that a new webserver can be created and that it
// implements the Server interface
func TestNewWebserver(t *testing.T) {
	log := &logrus.Entry{}
	cfg := testconfig.Get("", "", "", "", "")
	webserver := NewWebserver(cfg, log, http.Handler(nil))
	assert.Implements(t, (*Server)(nil), webserver)
}

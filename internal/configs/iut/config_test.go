// Copyright 2022 Axis Communications AB.
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
package config

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Test that it is possible to get a Cfg from Get with values taken from environment variables.
func TestGet(t *testing.T) {
	port := "8080"
	serverHost := "127.0.0.1"
	logLevel := "DEBUG"
	logFilePath := "path/to/a/file"
	timeoutStr := "1m"
	databaseHost := "etcd"
	databasePort := "12345"

	os.Setenv("SERVICE_HOST", serverHost)
	os.Setenv("SERVICE_PORT", port)
	os.Setenv("LOGLEVEL", logLevel)
	os.Setenv("LOG_FILE_PATH", logFilePath)
	os.Setenv("REQUEST_TIMEOUT", timeoutStr)
	os.Setenv("ETOS_ETCD_HOST", databaseHost)
	os.Setenv("ETOS_ETCD_PORT", databasePort)

	timeout, _ := time.ParseDuration(timeoutStr)

	conf, ok := Get().(*cfg)
	assert.Truef(t, ok, "cfg returned from get is not a config interface")
	assert.Equal(t, port, conf.servicePort)
	assert.Equal(t, serverHost, conf.serviceHost)
	assert.Equal(t, logLevel, conf.logLevel)
	assert.Equal(t, logFilePath, conf.logFilePath)
	assert.Equal(t, databaseHost, conf.databaseHost)
	assert.Equal(t, databasePort, conf.databasePort)
	assert.Equal(t, timeout, conf.timeout)
	assert.Equal(t, timeout, conf.timeout)
}

type getter func() string

// Test that the getters in the Cfg struct return the values from the struct.
func TestGetters(t *testing.T) {
	conf := &cfg{
		serviceHost:  "127.0.0.1",
		servicePort:  "8080",
		logLevel:     "TRACE",
		logFilePath:  "a/file/path.json",
		databaseHost: "etcd",
		databasePort: "12345",
	}
	tests := []struct {
		name     string
		cfg      *cfg
		function getter
		value    string
	}{
		{name: "ServiceHost", cfg: conf, function: conf.ServiceHost, value: conf.serviceHost},
		{name: "ServicePort", cfg: conf, function: conf.ServicePort, value: conf.servicePort},
		{name: "LogLevel", cfg: conf, function: conf.LogLevel, value: conf.logLevel},
		{name: "LogFilePath", cfg: conf, function: conf.LogFilePath, value: conf.logFilePath},
		{name: "DatabaseURI", cfg: conf, function: conf.DatabaseURI, value: fmt.Sprintf("%s:%s", conf.databaseHost, conf.databasePort)},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.value, testCase.function())
		})
	}
}

// TestTimeoutGetter tests the getter for Timeout. Similar to TestGetters, but since
// Timeout is not a "func() string" we separate its test.
func TestTimeoutGetter(t *testing.T) {
	timeout, _ := time.ParseDuration("1m")
	conf := &cfg{
		timeout: timeout,
	}
	assert.Equal(t, conf.timeout, conf.Timeout())
}

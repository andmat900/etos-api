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

package testconfig

import (
	config "github.com/eiffel-community/etos-api/internal/configs/base"
)

type cfg struct {
	serviceHost   string
	servicePort   string
	etosNamespace string
	logLevel      string
	logFilePath   string
}

// Get returns a new testconfig object/struct with the provided parameters
func Get(
	serviceHost,
	servicePort,
	etosNamespace,
	logLevel,
	logFilePath string,
) config.Config {
	return &cfg{
		serviceHost:   serviceHost,
		servicePort:   servicePort,
		etosNamespace: etosNamespace,
		logLevel:      logLevel,
		logFilePath:   logFilePath,
	}
}

// ServiceHost returns the Service Host testconfig parameter
func (c *cfg) ServiceHost() string {
	return c.serviceHost
}

// ServicePort returns the Service Port testconfig parameter
func (c *cfg) ServicePort() string {
	return c.servicePort
}

// LogLevel returns the Log level testconfig parameter
func (c *cfg) LogLevel() string {
	return c.logLevel
}

// LogFilePath returns the Log file path testconfig parameter
func (c *cfg) LogFilePath() string {
	return c.logFilePath
}

// etosNamespace returns the namespace testconfig parameter
func (c *cfg) ETOSNamespace() string {
	return c.etosNamespace
}

// DatabaseURI returns the URI to the ETOS database.
func (c *cfg) DatabaseURI() string {
	return "etcd-client:2379"
}

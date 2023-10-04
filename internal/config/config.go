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
package config

import (
	"flag"
	"os"
)

// Config interface for retreiving configuration options.
type Config interface {
	ServiceHost() string
	ServicePort() string
	LogLevel() string
	LogFilePath() string
	ETOSNamespace() string
}

// cfg implements the Config interface.
type cfg struct {
	serviceHost   string
	servicePort   string
	logLevel      string
	logFilePath   string
	etosNamespace string
}

// Get creates a config interface based on input parameters or environment variables.
func Get() Config {
	var conf cfg

	flag.StringVar(&conf.serviceHost, "address", EnvOrDefault("SERVICE_HOST", "127.0.0.1"), "Address to serve API on")
	flag.StringVar(&conf.servicePort, "port", EnvOrDefault("SERVICE_PORT", "8080"), "Port to serve API on")
	flag.StringVar(&conf.logLevel, "loglevel", EnvOrDefault("LOGLEVEL", "INFO"), "Log level (TRACE, DEBUG, INFO, WARNING, ERROR, FATAL, PANIC).")
	flag.StringVar(&conf.logFilePath, "logfilepath", os.Getenv("LOG_FILE_PATH"), "Path, including filename, for the log files to create.")
	flag.StringVar(&conf.etosNamespace, "etosnamespace", ReadNamespaceOrEnv("ETOS_NAMESPACE"), "Path, including filename, for the log files to create.")

	flag.Parse()
	return &conf
}

// ServiceHost returns the host of the service.
func (c *cfg) ServiceHost() string {
	return c.serviceHost
}

// ServicePort returns the port of the service.
func (c *cfg) ServicePort() string {
	return c.servicePort
}

// LogLevel returns the log level.
func (c *cfg) LogLevel() string {
	return c.logLevel
}

// LogFilePath returns the path to where log files should be stored, including filename.
func (c *cfg) LogFilePath() string {
	return c.logFilePath
}

// ETOSNamespace returns the ETOS namespace.
func (c *cfg) ETOSNamespace() string {
	return c.etosNamespace
}

// EnvOrDefault will look up key in environment variables and return if it exists, else return the fallback value.
func EnvOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// ReadNamespaceOrEnv checks if there's a nemspace file inside the container, else returns
// environment variable with envKey as name.
func ReadNamespaceOrEnv(envKey string) string {
	inClusterNamespace, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return os.Getenv(envKey)
	}
	return string(inClusterNamespace)
}

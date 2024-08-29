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
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// Config interface for retrieving configuration options.
type Config interface {
	ServiceHost() string
	ServicePort() string
	LogLevel() string
	LogFilePath() string
	Timeout() time.Duration
	EventRepositoryHost() string
	IutWaitTimeoutHard() time.Duration
	IutWaitTimeoutSoft() time.Duration
	DatabaseURI() string
}

// cfg implements the Config interface.
type cfg struct {
	serviceHost         string
	servicePort         string
	logLevel            string
	logFilePath         string
	timeout             time.Duration
	databaseHost        string
	databasePort        string
	eventRepositoryHost string
	iutWaitTimeoutHard  time.Duration
	iutWaitTimeoutSoft  time.Duration
}

// Get creates a config interface based on input parameters or environment variables.
func Get() Config {
	var conf cfg

	defaultTimeout, err := time.ParseDuration(EnvOrDefault("REQUEST_TIMEOUT", "1m"))
	if err != nil {
		logrus.Panic(err)
	}

	iutWaitTimeoutHard, err := time.ParseDuration(EnvOrDefault("IUT_WAIT_TIMEOUT", "1h"))
	if err != nil {
		logrus.Panic(err)
	}

	iutWaitTimeoutSoft, err := time.ParseDuration(EnvOrDefault("IUT_WAIT_TIMEOUT_SOFT", "30m"))
	if err != nil {
		logrus.Panic(err)
	}

	flag.StringVar(&conf.serviceHost, "address", EnvOrDefault("SERVICE_HOST", "127.0.0.1"), "Address to serve API on")
	flag.StringVar(&conf.servicePort, "port", EnvOrDefault("SERVICE_PORT", "8080"), "Port to serve API on")
	flag.StringVar(&conf.logLevel, "loglevel", EnvOrDefault("LOGLEVEL", "INFO"), "Log level (TRACE, DEBUG, INFO, WARNING, ERROR, FATAL, PANIC).")
	flag.StringVar(&conf.logFilePath, "logfilepath", os.Getenv("LOG_FILE_PATH"), "Path, including filename, for the log files to create.")
	flag.DurationVar(&conf.timeout, "timeout", defaultTimeout, "Maximum timeout for requests to Provider Service.")
	flag.StringVar(&conf.databaseHost, "database_host", EnvOrDefault("ETOS_ETCD_HOST", "etcd-client"), "Host to ETOS database")
	flag.StringVar(&conf.databasePort, "database_port", EnvOrDefault("ETOS_ETCD_PORT", "2379"), "Port to ETOS database")
	flag.StringVar(&conf.eventRepositoryHost, "eventrepository_url", os.Getenv("ETOS_GRAPHQL_SERVER"), "URL to the GraphQL server to use for event lookup.")
	flag.DurationVar(&conf.iutWaitTimeoutHard, "hard iut wait timeout", iutWaitTimeoutHard, "Hard wait timeout for IUT checkout")
	flag.DurationVar(&conf.iutWaitTimeoutSoft, "soft iut wait timeout", iutWaitTimeoutSoft, "Soft wait timeout for IUT checkout")
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

// Timeout returns the request timeout for Provider Service API.
func (c *cfg) Timeout() time.Duration {
	return c.timeout
}

// EventRepositoryHost returns the host to use for event lookups.
func (c *cfg) EventRepositoryHost() string {
	return c.eventRepositoryHost
}

// IutWaitTimeoutHard returns the hard timeout for IUT checkout
func (c *cfg) IutWaitTimeoutHard() time.Duration {
	return c.iutWaitTimeoutHard
}

// IutWaitTimeoutSoft returns the soft timeout for IUT checkout
func (c *cfg) IutWaitTimeoutSoft() time.Duration {
	return c.iutWaitTimeoutSoft
}

// DatabaseURI returns the URI to the ETOS database.
func (c *cfg) DatabaseURI() string {
	return fmt.Sprintf("%s:%s", c.databaseHost, c.databasePort)
}

// EnvOrDefault will look up key in environment variables and return if it exists, else return the fallback value.
func EnvOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

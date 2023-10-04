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
package logging

import (
	"github.com/sirupsen/logrus"
)

// Setup sets up logging to file with a JSON format and to stdout in text format.
func Setup(loglevel string, hooks []logrus.Hook) (*logrus.Logger, error) {
	log := logrus.New()

	logLevel, err := logrus.ParseLevel(loglevel)
	if err != nil {
		return log, err
	}
	for _, hook := range hooks {
		log.AddHook(hook)
	}

	log.SetLevel(logLevel)
	log.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	log.SetReportCaller(true)
	return log, nil
}

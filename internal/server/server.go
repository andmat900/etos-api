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
package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/eiffel-community/etos-api/internal/config"
	"github.com/sirupsen/logrus"
)

// Server interface for serving up the service.
type Server interface {
	Start() error
	Close(ctx context.Context) error
}

// WebService is a struct for webservices implementing the Server interface.
type WebService struct {
	server *http.Server
	cfg    config.Config
	logger *logrus.Entry
}

// NewWebService creates a new Server of the webservice type.
func NewWebService(cfg config.Config, log *logrus.Entry, handler http.Handler) Server {
	webservice := &WebService{
		server: &http.Server{
			Addr:    fmt.Sprintf("%s:%s", cfg.ServiceHost(), cfg.ServicePort()),
			Handler: handler,
		},
		cfg:    cfg,
		logger: log,
	}
	return webservice
}

// Start a webservice and block until closed or crashed.
func (s *WebService) Start() error {
	s.logger.Infof("Starting webservice listening on %s:%s", s.cfg.ServiceHost(), s.cfg.ServicePort())
	return s.server.ListenAndServe()
}

// Close calls shutdown on the webservice. Shutdown times out if context is cancelled.
func (s *WebService) Close(ctx context.Context) error {
	s.logger.Info("Shutting down webservice")
	return s.server.Shutdown(ctx)
}

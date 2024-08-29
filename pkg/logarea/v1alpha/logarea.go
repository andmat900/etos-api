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
package logarea

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"runtime"
	"time"

	config "github.com/eiffel-community/etos-api/internal/configs/logarea"
	"github.com/eiffel-community/etos-api/pkg/application"
	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// REGEX for matching /testrun/tercc-id/suite/main-suite-id/subsuite/subsuite-id/suite.
const (
	REGEX   = "/testrun/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/suite/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/subsuite/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/suite"
	TIMEOUT = 15 * time.Second
)

type LogAreaApplication struct {
	logger *logrus.Entry
	cfg    config.Config
	client *clientv3.Client
	regex  *regexp.Regexp
}

type LogAreaHandler struct {
	logger *logrus.Entry
	cfg    config.Config
	client *clientv3.Client
	regex  *regexp.Regexp
}

// Close cancels the application context and closes the ETCD client.
func (a *LogAreaApplication) Close() {
	a.client.Close()
}

// New returns a new LogAreaApplication object/struct.
func New(cfg config.Config, log *logrus.Entry) application.Application {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{cfg.DatabaseURI()},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.WithError(err).Fatal("Failed to create etcd client")
		return nil
	}
	// MustCompile panics if the regular expression cannot be compiled.
	// Since the regular expression is hard-coded, it should never fail in production.
	regex := regexp.MustCompile(REGEX)
	return &LogAreaApplication{
		logger: log,
		cfg:    cfg,
		client: cli,
		regex:  regex,
	}
}

// LoadRoutes loads all the v1alpha routes.
func (a LogAreaApplication) LoadRoutes(router *httprouter.Router) {
	handler := &LogAreaHandler{a.logger, a.cfg, a.client, a.regex}
	router.GET("/v1alpha/selftest/ping", handler.Selftest)
	router.GET("/v1alpha/logarea/:identifier", handler.panicRecovery(handler.timeoutHandler(handler.GetFileURLs)))
}

// Selftest is a handler to just return 204.
func (h LogAreaHandler) Selftest(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusNoContent)
}

type Response map[string]Directory

type Downloadable struct {
	URL  string       `json:"url"`
	Name []FilterType `json:"name"`
}

type Directory struct {
	Logs      []Downloadable `json:"logs"`
	Artifacts []Downloadable `json:"artifacts"`
}

// getDownloadURLs will request the log area and get the URLs for the artifacts and logs, running a filter over them.
func (h LogAreaHandler) getDownloadURLs(ctx context.Context, logger *logrus.Entry, subSuite []byte, download Download) (logs []Downloadable, artifacts []Downloadable, err error) {
	response, err := download.Request.Do(ctx, logger)
	if err != nil {
		logger.Errorf("failed to request URLs from logarea: %s", download.Request.URL)
		return nil, nil, err
	}
	defer response.Body.Close()
	jsondata, err := io.ReadAll(response.Body)
	if err != nil {
		logger.Errorf("failed to read response body from logarea: %s", download.Request.URL)
		return nil, nil, err
	}
	logUrls, err := download.Filters.Logs.Run(jsondata, response.Header, subSuite, download.Filters.BaseURL)
	if err != nil {
		logger.WithError(err).Error("could not run filters on log URLs")
		return nil, nil, err
	}
	artifactUrls, err := download.Filters.Artifacts.Run(jsondata, response.Header, subSuite, download.Filters.BaseURL)
	if err != nil {
		logger.Error("could not run filters on artifact URLs")
		return nil, nil, err
	}
	return logUrls, artifactUrls, nil
}

// GetFileURLs is an endpoint for getting file URLs from a log area.
func (h LogAreaHandler) GetFileURLs(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*10)
	defer cancel()
	directories := make(Response)
	identifier := ps.ByName("identifier")
	// Making it possible for us to correlate logs to a specific connection
	logger := h.logger.WithField("identifier", identifier)
	w.Header().Set("X-Content-Type-Options", "nosniff")

	response, err := h.client.Get(ctx, fmt.Sprintf("/testrun/%s/suite", identifier), clientv3.WithPrefix())
	if err != nil {
		logger.WithError(err).Error("Failed to get file URLs")
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Add("Retry-After", "10")
		return
	}
	for _, ev := range response.Kvs {
		// Verify that 'ev.Value' is an actual sub suite definition and not another
		// field in the ETCD database. Since we are prefix searching on /testrun/suiteid/suite
		// it is very possible that more data will arrive than we are interested in.
		if !h.regex.Match(ev.Key) {
			continue
		}
		suite := Suite{}
		if err := json.Unmarshal(ev.Value, &suite); err != nil {
			logger.WithError(err).Error("Failed to unmarshal suite")
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Add("Retry-After", "10")
			return
		}
		logUrls := []Downloadable{}
		artifactUrls := []Downloadable{}
		for _, download := range suite.LogArea.Download {
			logs, artifacts, err := h.getDownloadURLs(ctx, logger, ev.Value, download)
			if err != nil {
				logger.WithError(err).Error("Failed to download")
				w.WriteHeader(http.StatusInternalServerError)
				w.Header().Add("Retry-After", "10")
				return
			}
			logUrls = append(logUrls, logs...)
			artifactUrls = append(artifactUrls, artifacts...)
		}
		directories[suite.Name] = Directory{Logs: logUrls, Artifacts: artifactUrls}
	}

	resp, _ := json.Marshal(directories) //nolint:errchkjson
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}

// timeoutHandler will change the request context to a timeout context.
func (h LogAreaHandler) timeoutHandler(
	fn func(http.ResponseWriter, *http.Request, httprouter.Params),
) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), TIMEOUT)
		defer cancel()
		newRequest := r.WithContext(ctx)
		fn(w, newRequest, ps)
	}
}

// panicRecovery tracks panics from the service, logs them and returns an error response to the user.
func (h LogAreaHandler) panicRecovery(
	fn func(http.ResponseWriter, *http.Request, httprouter.Params),
) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		defer func() {
			if err := recover(); err != nil {
				buf := make([]byte, 2048)
				n := runtime.Stack(buf, false)
				buf = buf[:n]
				h.logger.WithField(
					"identifier", ps.ByName("identifier"),
				).WithContext(
					r.Context(),
				).Errorf("recovering from err %+v\n %s", err, buf)
				identifier := ps.ByName("identifier")

				response, _ := json.Marshal(fmt.Sprintf("unknown error: contact server admin with id '%s'", identifier))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write(response)
			}
		}()
		fn(w, r, ps)
	}
}

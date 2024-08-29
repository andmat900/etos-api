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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"sync"

	config "github.com/eiffel-community/etos-api/internal/configs/iut"
	"github.com/eiffel-community/etos-api/internal/iut/checkoutable"
	"github.com/eiffel-community/etos-api/internal/iut/contextmanager"
	"github.com/eiffel-community/etos-api/internal/iut/responses"
	"github.com/eiffel-community/etos-api/pkg/iut/application"
	"github.com/package-url/packageurl-go"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/eiffel-community/eiffelevents-sdk-go"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
)

var (
	service_version string
)

// BASEREGEX for matching /testrun/tercc-id/provider/iuts/reference.
const BASEREGEX = "/testrun/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/provider/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/iuts"

const EtcdTreePathPrefix = "/purl"

type V1Alpha1Application struct {
	logger   *logrus.Entry
	cfg      config.Config
	database *clientv3.Client
	cm       *contextmanager.ContextManager
	wg       *sync.WaitGroup
}

type V1Alpha1Handler struct {
	logger   *logrus.Entry
	cfg      config.Config
	database *clientv3.Client
	cm       *contextmanager.ContextManager
	wg       *sync.WaitGroup
}

type StartRequest struct {
	MinimumAmount     int                                                 `json:"minimum_amount"`
	MaximumAmount     int                                                 `json:"maximum_amount"`
	ArtifactIdentity  string                                              `json:"identity"`
	ArtifactID        string                                              `json:"artifact_id"`
	ArtifactCreated   eiffelevents.ArtifactCreatedV3                      `json:"artifact_created,omitempty"`
	ArtifactPublished eiffelevents.ArtifactPublishedV3                    `json:"artifact_published,omitempty"`
	TERCC             eiffelevents.TestExecutionRecipeCollectionCreatedV4 `json:"tercc,omitempty"`
	Context           uuid.UUID                                           `json:"context,omitempty"`
	Dataset           Dataset                                             `json:"dataset,omitempty"`
}

type Dataset struct {
	Greed interface{} `json:"greed"`
}

type StartResponse struct {
	Id uuid.UUID `json:"id"`
}

type statusResponse struct {
	Id          uuid.UUID `json:"id"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
}

type StopRequest []*checkoutable.Iut

type StatusRequest struct {
	Id uuid.UUID `json:"id"`
}

// Close does nothing atm. Present for interface coherence
func (a *V1Alpha1Application) Close() {
	a.wg.Wait()
}

// New returns a new V1Alpha1Application object/struct
func New(cfg config.Config, log *logrus.Entry, ctx context.Context, cm *contextmanager.ContextManager, cli *clientv3.Client) application.Application {
	return &V1Alpha1Application{
		logger:   log,
		cfg:      cfg,
		database: cli,
		cm:       cm,
		wg:       &sync.WaitGroup{},
	}
}

// LoadRoutes loads all the v1alpha1 routes.
func (a V1Alpha1Application) LoadRoutes(router *httprouter.Router) {
	handler := &V1Alpha1Handler{a.logger, a.cfg, a.database, a.cm, a.wg}
	router.GET("/v1alpha1/selftest/ping", handler.Selftest)
	router.POST("/start", handler.panicRecovery(handler.timeoutHandler(handler.Start)))
	router.GET("/status", handler.panicRecovery(handler.timeoutHandler(handler.Status)))
	router.POST("/stop", handler.panicRecovery(handler.timeoutHandler(handler.Stop)))
}

// Selftest is a handler to just return 204.
func (h V1Alpha1Handler) Selftest(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	responses.RespondWithError(w, http.StatusNoContent, "")
}

func (h V1Alpha1Handler) readPurlFromEtcd(id uuid.UUID) (*packageurl.PackageURL, error) {
	ctx := context.Background()
	key := fmt.Sprintf("%s/%s", EtcdTreePathPrefix, id)
	response, err := h.database.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if len(response.Kvs) == 0 {
		return nil, fmt.Errorf("Purl not found for id: %s", id)
	}
	purl, err := packageurl.FromString(string(response.Kvs[0].Value))
	return &purl, nil
}

func (h V1Alpha1Handler) savePurlToEtcd(id uuid.UUID, purl packageurl.PackageURL, maximumAmount int) error {
	ctx := context.Background()
	key := fmt.Sprintf("%s/%s", EtcdTreePathPrefix, id)
	_, err := h.database.Put(ctx, key, purl.ToString())
	if err != nil {
		return err
	}
	return ctx.Err()
}

// removePurlFromEtcd removes the purl from the given id from the database
func (h V1Alpha1Handler) removePurlFromEtcd(id string) error {
	ctx := context.Background()
	key := fmt.Sprintf("%s/%s", EtcdTreePathPrefix, id)
	_, err := h.database.Delete(ctx, key)
	if err != nil {
		return err
	}
	return ctx.Err()
}

// getPackageUrl returns a dummy instance of PackageURL
func (h V1Alpha1Handler) getPackageUrl(id string) (packageurl.PackageURL, error) {
	purl, err := packageurl.FromString("http://example.com/testpackage@1.0.0")
	return purl, err
}

// Start handles the start request
func (h V1Alpha1Handler) Start(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx := context.Background()
	checkOutID := uuid.New()
	identifier := r.Header.Get("X-Etos-Id")
	logger := h.logger.WithField("identifier", identifier).WithContext(ctx)

	startReq, _ := h.verifyStartInput(ctx, logger, r)
	purl, _ := h.getPackageUrl(startReq.ArtifactID)
	err := h.savePurlToEtcd(checkOutID, purl, startReq.MaximumAmount)
	if err != nil {
		responses.RespondWithError(w, http.StatusInternalServerError, err.Error())
	}
	responses.RespondWithJSON(w, http.StatusOK, StartResponse{Id: checkOutID})
}

// Status handles the status request
func (h V1Alpha1Handler) Status(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx := context.Background()
	statusReq, _ := h.verifyStatusInput(ctx, r)
	_, err := h.readPurlFromEtcd(statusReq.Id)
	if err != nil {
		responses.RespondWithError(w, http.StatusInternalServerError, err.Error())
	}
	status := statusResponse{
		Id:          statusReq.Id,
		Status:      "OK",
		Description: "Status ok",
	}
	responses.RespondWithJSON(w, http.StatusOK, status)
}

// Stop removes the purl from the database and returns a stop response
func (h V1Alpha1Handler) Stop(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	iuts, _ := h.verifyStopInput(context.Background(), r)
	for _, iut := range iuts {
		h.removePurlFromEtcd(iut.ArtifactId)
	}
	responses.RespondWithJSON(w, http.StatusNoContent, "")
}

// timeoutHandler will change the request context to a timeout context.
func (h V1Alpha1Handler) timeoutHandler(
	fn func(http.ResponseWriter, *http.Request, httprouter.Params),
) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), h.cfg.Timeout())
		defer cancel()
		newRequest := r.WithContext(ctx)
		fn(w, newRequest, ps)
	}
}

// verifyStartInput verify input (json body) from a start request
func (h V1Alpha1Handler) verifyStartInput(ctx context.Context, logger *logrus.Entry, r *http.Request) (StartRequest, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return StartRequest{}, NewHTTPError(
			fmt.Errorf("failed read request body - Reason: %s", err.Error()),
			http.StatusBadRequest,
		)
	}
	defer r.Body.Close()

	request, err := h.tryLoadStartRequest(ctx, logger, body)
	if err != nil {
		return request, NewHTTPError(err, http.StatusBadRequest)
	}

	if request.ArtifactID == "" || request.ArtifactIdentity == "" {
		return request, NewHTTPError(
			errors.New("both 'artifact_identity' and 'artifact_id' are required"),
			http.StatusBadRequest,
		)
	}

	if request.MinimumAmount == 0 {
		return request, NewHTTPError(
			errors.New("minimumAmount parameter is mandatory"),
			http.StatusBadRequest,
		)
	}

	_, purlErr := packageurl.FromString(request.ArtifactIdentity)
	if purlErr != nil {
		return request, NewHTTPError(purlErr, http.StatusBadRequest)
	}

	return request, ctx.Err()
}

// verifyStatusInput verify input (url parameters) from the status request
func (h V1Alpha1Handler) verifyStatusInput(ctx context.Context, r *http.Request) (StatusRequest, error) {
	id, err := uuid.Parse(r.URL.Query().Get("id"))
	if err != nil {
		return StatusRequest{}, NewHTTPError(
			fmt.Errorf("Error parsing id parameter in status request - Reason: %s", err.Error()),
			http.StatusBadRequest)
	}
	request := StatusRequest{Id: id}

	return request, ctx.Err()
}

// verifyStopInput verify input (json body) from the stop request
func (h V1Alpha1Handler) verifyStopInput(ctx context.Context, r *http.Request) (StopRequest, error) {
	request := StopRequest{}
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return request, NewHTTPError(fmt.Errorf("unable to decode post body %+v", err), http.StatusBadRequest)
	}
	return request, ctx.Err()
}

// sendError sends an error HTTP response depending on which error has been returned.
func sendError(w http.ResponseWriter, err error) {
	httpError, ok := err.(*HTTPError)
	if !ok {
		responses.RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("unknown error %+v", err))
	} else {
		responses.RespondWithError(w, httpError.Code, httpError.Message)
	}
}

// panicRecovery tracks panics from the service, logs them and returns an error response to the user.
func (h V1Alpha1Handler) panicRecovery(
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
				responses.RespondWithError(
					w,
					http.StatusInternalServerError,
					fmt.Sprintf("unknown error: contact server admin with id '%s'", identifier),
				)
			}
		}()
		fn(w, r, ps)
	}
}

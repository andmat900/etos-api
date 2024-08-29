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

// Package checkoutable handles checkout/checkin operations
package checkoutable

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Iut struct {
	ProviderId string      `json:"provider_id,omitempty"`
	Identity   string      `json:"identity"`
	ArtifactId string      `json:"artifact_id"`
	Reference  string      `json:"reference"`
	TestRunner interface{} `json:"test_runner,omitempty"`
	logger     *logrus.Entry
}

// Fulfill json marshall interface for StatusResponses
func (s Iut) MarshalBinary() ([]byte, error) {
	return json.Marshal(s)
}

// NewIut creates a new iut struct
func NewIut(identity string, artifactId string, logger *logrus.Entry) *Iut {
	return &Iut{
		Identity:   identity,
		ArtifactId: artifactId,
		logger:     logger,
	}
}

// AddLogger adds a logger to the iut struct, can be used if iut is created without NewIut
func (i *Iut) AddLogger(logger *logrus.Entry) {
	i.logger = logger
}

// Checkout checks out an IUT.
func (i *Iut) Checkout(ctx context.Context, configurationName string, eiffelContext uuid.UUID) error {
	i.logger.Infof("checking out IUT %s, reference: %s", i.Identity, i.Reference)
	return ctx.Err()
}

// CheckIn checks in IUT
func (i *Iut) CheckIn(withForce bool) ([]string, error) {
	// We shall ALWAYS try to check in IUTs if requested.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	if i.Reference == "" {
		return nil, ctx.Err()
	}
	var references []string
	return references, ctx.Err()
}

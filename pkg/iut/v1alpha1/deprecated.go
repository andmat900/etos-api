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
	"fmt"

	"github.com/eiffel-community/eiffelevents-sdk-go"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type DeprecatedStartRequest struct {
	MinimumAmount     int                                                 `json:"minimum_amount"`
	MaximumAmount     int                                                 `json:"maximum_amount"`
	ArtifactIdentity  string                                              `json:"identity"`
	ArtifactID        string                                              `json:"artifact_id"`
	ArtifactCreated   eiffelevents.ArtifactCreatedV3                      `json:"artifact_created,omitempty"`
	ArtifactPublished eiffelevents.ArtifactPublishedV3                    `json:"artifact_published"`
	TERCC             eiffelevents.TestExecutionRecipeCollectionCreatedV4 `json:"tercc,omitempty"`
	Context           uuid.UUID                                           `json:"context,omitempty"`
	Dataset           DeprecatedDataset                                   `json:"dataset,omitempty"`
}

type DeprecatedDataset struct {
	Greed interface{} `json:"greed"`
}

// startRequestFromDeprecatedStartRequest will take a DeprecatedStartRequest struct and with the power
// of ER will create a StartRequest
func (h V1Alpha1Handler) startRequestFromDeprecatedStartRequest(ctx context.Context, logger *logrus.Entry, deprecatedStartRequest DeprecatedStartRequest) (StartRequest, error) {
	request := StartRequest{
		MinimumAmount:     deprecatedStartRequest.MinimumAmount,
		MaximumAmount:     deprecatedStartRequest.MaximumAmount,
		ArtifactIdentity:  deprecatedStartRequest.ArtifactIdentity,
		ArtifactID:        deprecatedStartRequest.ArtifactID,
		ArtifactCreated:   deprecatedStartRequest.ArtifactCreated,
		ArtifactPublished: deprecatedStartRequest.ArtifactPublished,
		TERCC:             deprecatedStartRequest.TERCC,
		Context:           deprecatedStartRequest.Context,
		Dataset: Dataset{
			Greed: deprecatedStartRequest.Dataset.Greed,
		},
	}
	return request, nil
}

// tryLoadNewStartRequest will attempt to load a []byte into the StartRequest struct
func (h V1Alpha1Handler) tryLoadNewStartRequest(ctx context.Context, logger *logrus.Entry, data []byte) (StartRequest, error) {
	request := StartRequest{}
	//er := eventrepository.NewER(h.cfg, logger)
	err := json.Unmarshal(data, &request)
	if err != nil {
		return request, err
	}
	return request, nil
}

// tryLoadStartRequest tries to first load the StartRequest, if that fails
// it will try to load the old, deprecated, version.
func (h V1Alpha1Handler) tryLoadStartRequest(ctx context.Context, logger *logrus.Entry, data []byte) (StartRequest, error) {
	request, err := h.tryLoadNewStartRequest(ctx, logger, data)
	if err == nil {
		return request, nil
	}

	deprecatedRequest := DeprecatedStartRequest{}
	if err := json.Unmarshal(data, &deprecatedRequest); err != nil {
		return request, fmt.Errorf("unable to decode post body - Reason: %s", err.Error())
	}
	return h.startRequestFromDeprecatedStartRequest(ctx, logger, deprecatedRequest)
}

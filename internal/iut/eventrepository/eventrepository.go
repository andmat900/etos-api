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
package eventrepository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	config "github.com/eiffel-community/etos-api/internal/configs/iut"

	"github.com/eiffel-community/eiffelevents-sdk-go"
	"github.com/machinebox/graphql"
	"github.com/sirupsen/logrus"
)

// EventRepository is the interface used no matter which event repository is used.
type EventRepository interface {
	GetArtifactByID(context.Context, string) (eiffelevents.ArtifactCreatedV3, error)
	GetArtifactByIdentity(context.Context, string) (eiffelevents.ArtifactCreatedV3, error)
	GetArtifactPublishedByArtifactCreatedID(ctx context.Context, id string) (eiffelevents.ArtifactPublishedV3, error)
}

type artCResponse struct {
	Items []eiffelevents.ArtifactCreatedV3 `json:"items,omitempty"`
}

type artPResponse struct {
	Items []eiffelevents.ArtifactPublishedV3 `json:"items,omitempty"`
}

// ER implements EventRepository
type ER struct {
	cfg    config.Config
	logger *logrus.Entry
}

// NewER creates a new ER client implementing the EventRepository interface.
func NewER(cfg config.Config, logger *logrus.Entry) EventRepository {
	return &ER{
		cfg:    cfg,
		logger: logger,
	}
}

// GetArtifactByID uses the 'meta.id' key in artifact created events to find
// the artifact representation in ER.
func (er *ER) GetArtifactByID(ctx context.Context, id string) (eiffelevents.ArtifactCreatedV3, error) {
	query := map[string]string{"meta.id": id, "meta.type": "EiffelArtifactCreatedEvent", "pageSize": "1"}
	body, err := er.getEvents(ctx, query)
	if err != nil {
		return eiffelevents.ArtifactCreatedV3{}, err
	}
	var event artCResponse
	if err = json.Unmarshal(body, &event); err != nil {
		return eiffelevents.ArtifactCreatedV3{}, err
	}
	if len(event.Items) == 0 {
		return eiffelevents.ArtifactCreatedV3{}, errors.New("no artifact created event found")
	}
	return event.Items[0], ctx.Err()
}

// GetArtifactPublishedByArtifactCreatedID uses the 'meta.id' from an artifact created events to find
// the linked artifact published event in the Eventrepository.
func (er *ER) GetArtifactPublishedByArtifactCreatedID(ctx context.Context, id string) (eiffelevents.ArtifactPublishedV3, error) {
	query := map[string]string{"links.target": id, "meta.type": "EiffelArtifactPublishedEvent", "links.type": "ARTIFACT", "pageSize": "1"}
	body, err := er.getEvents(ctx, query)
	if err != nil {
		return eiffelevents.ArtifactPublishedV3{}, err
	}
	var event artPResponse
	if err = json.Unmarshal(body, &event); err != nil {
		return eiffelevents.ArtifactPublishedV3{}, err
	}
	if len(event.Items) == 0 {
		return eiffelevents.ArtifactPublishedV3{}, errors.New("no artifact published event found")
	}
	return event.Items[0], ctx.Err()
}

// getEvents creates an eventrepository event query and sends it to the GoER eventrepository.
func (er *ER) getEvents(ctx context.Context, query map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/v1/events", er.cfg.EventRepositoryHost()), nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	for key, value := range query {
		q.Add(key, value)
	}
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if response.StatusCode == 404 {
		return nil, errors.New("event not found in event repository")
	}

	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return body, ctx.Err()
}

// GetArtifactByIdentity uses the 'data.identity' key in artifact created events to find
// the artifact representation in ER.
func (er *ER) GetArtifactByIdentity(ctx context.Context, identity string) (eiffelevents.ArtifactCreatedV3, error) {
	return er.artifact(ctx, fmt.Sprintf("{'data.identity': {'$regex': '^%s'}}", identity))
}

// ArtifactQuery is a generic query for finding artifact created identity and id.
// It requires a searchString to be added as a parameter.
var ArtifactQuery = `
query ArtifactQuery($searchString: String) {
	artifactCreated(search: $searchString, last: 1) {
		edges {
			node {
				data {
					identity
				}
				meta {
					id
				}
			}
		}
	}
}
`

// Edge holds the Artifact node.
type Edge struct {
	Node eiffelevents.ArtifactCreatedV3 `json:"node"`
}

// GraphQLArtifact is a slice of Edges that represent artifacts.
type GraphQLArtifact struct {
	Edges []Edge `json:"edges"`
}

// GraphQLResponse is the response from the GraphQL event repository.
type GraphQLResponse struct {
	ArtifactCreated GraphQLArtifact `json:"artifactCreated"`
}

// NoArtifact indicates that no artifacts were found.
type NoArtifact struct {
	Message string
}

// Error returns the string representation of the NoArtifact error.
func (e *NoArtifact) Error() string {
	return e.Message
}

// artifact makes a request against the event repository with the searchString provided.
func (er *ER) artifact(ctx context.Context, searchString string) (eiffelevents.ArtifactCreatedV3, error) {
	request := graphql.NewRequest(ArtifactQuery)
	request.Var("searchString", searchString)

	var response GraphQLResponse
	gqlClient := graphql.NewClient(fmt.Sprintf("%s/graphql", er.cfg.EventRepositoryHost()))
	if err := gqlClient.Run(ctx, request, &response); err != nil {
		return eiffelevents.ArtifactCreatedV3{}, err
	}
	edges := response.ArtifactCreated.Edges
	if len(edges) == 0 {
		return eiffelevents.ArtifactCreatedV3{}, &NoArtifact{fmt.Sprintf("no artifact returned for %s", searchString)}
	}
	// The query is limited to, at most, one response and we test that it is not zero just before this
	// which makes it safe to get the 0 index from edges.
	return edges[0].Node, ctx.Err()
}

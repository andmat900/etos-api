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
package events

import (
	"encoding/json"
	"fmt"
	"io"
)

// Event is a struct for SSE events as JSON since the ESR returns it as JSON.
type Event struct {
	ID       int    `json:"id"`
	Event    string `json:"event"`
	Data     string
	JSONData interface{} `json:"data"`
}

// New creates a new Event from a slice of bytes
func New(data []byte) (Event, error) {
	e := Event{}
	if err := json.Unmarshal(data, &e); err != nil {
		return e, err
	}
	data, err := json.Marshal(e.JSONData)
	if err != nil {
		return e, err
	}
	e.Data = string(data)
	return e, nil

}

// Write an event to a writer object
func (e Event) Write(w io.Writer) error {
	message := ""
	if e.ID != 0 {
		message += fmt.Sprintf("id: %d\n", e.ID)
	}
	if len(e.Event) > 0 {
		message += fmt.Sprintf("event: %s\n", e.Event)
	}
	message += fmt.Sprintf("data: %s\n\n", e.Data)
	_, err := w.Write([]byte(message))
	return err
}

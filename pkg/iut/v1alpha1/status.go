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

	"github.com/eiffel-community/etos-api/internal/iut/checkoutable"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Status struct {
	Status        string              `json:"status"`
	Description   string              `json:"description"`
	Iuts          []*checkoutable.Iut `json:"iuts"`
	IutReferences []string            `json:"iut_references"`
	baseKey       string
	iutsKey       string
	statusKey     string
	db            *clientv3.Client
}

// NewStatus will return a new status field. Does not automatically load data from database.
func NewStatus(baseKey string, id string, db *clientv3.Client) *Status {
	return &Status{
		baseKey:   baseKey,
		iutsKey:   fmt.Sprintf("%s/provider/%s/iuts", baseKey, id),
		statusKey: fmt.Sprintf("%s/provider/%s/status", baseKey, id),
		db:        db,
	}
}

// MarhslBinary fulfills json marshall interface for StatusResponses
func (s Status) MarshalBinary() ([]byte, error) {
	return json.Marshal(s)
}

// Pending sets status to PENDING and saves it to database.
func (s *Status) Pending(ctx context.Context, description string) error {
	s.Status = "PENDING"
	return s.Save(ctx)
}

// Failed sets status to FAILED and saves it to database.
func (s *Status) Failed(ctx context.Context, description string) error {
	s.Status = "FAILED"
	return s.Save(ctx)
}

// Done sets status to DONE and saves it to database.
func (s *Status) Done(ctx context.Context, description string) error {
	s.Status = "DONE"
	return s.Save(ctx)
}

// AddIUT adds a IUT to Status and saves it to database.
func (s *Status) AddIUT(ctx context.Context, iut *checkoutable.Iut) error {
	if err := s.addIut(ctx, iut); err != nil {
		return err
	}
	return s.Save(ctx)
}

// IUT returns an IUT from the database based on a checkout reference.
func (s Status) Iut(ctx context.Context, reference string) (*checkoutable.Iut, error) {
	key := fmt.Sprintf("%s/%s", s.iutsKey, reference)
	response, err := s.db.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if len(response.Kvs) == 0 {
		return nil, ctx.Err()
	}
	value := response.Kvs[0].Value
	var iut checkoutable.Iut
	err = json.Unmarshal(value, &iut)
	if err != nil {
		return nil, err
	}
	return &iut, ctx.Err()
}

// deleteReference deletes a reference from the iutReference slice.
func (s *Status) deleteReference(iut *checkoutable.Iut) {
	for index, reference := range s.IutReferences {
		if reference == iut.Reference {
			newSlice := make([]string, 0)
			newSlice = append(newSlice, s.IutReferences[:index]...)
			s.IutReferences = append(newSlice, s.IutReferences[index+1:]...)
		}
	}
}

// deleteIut deletes an IUT from the iuts slice.
func (s *Status) deleteIut(iutToRemove *checkoutable.Iut) {
	for index, iut := range s.Iuts {
		if iut.Reference == iutToRemove.Reference {
			newSlice := make([]*checkoutable.Iut, 0)
			newSlice = append(newSlice, s.Iuts[:index]...)
			s.Iuts = append(newSlice, s.Iuts[index+1:]...)
		}
	}
}

// RemoveIUT removes an IUT from the database and struct and saves it to database.
func (s *Status) RemoveIUT(ctx context.Context, iut *checkoutable.Iut) error {
	if err := s.removeIut(ctx, iut); err != nil {
		return err
	}
	s.deleteIut(iut)
	return s.Save(ctx)
}

// addIut adds an IUT to the struct and saves it to database.
func (s Status) addIut(ctx context.Context, iut *checkoutable.Iut) error {
	data, err := iut.MarshalBinary()
	if err != nil {
		return err
	}
	_, err = s.db.Put(ctx, fmt.Sprintf("%s/%s", s.iutsKey, iut.Reference), string(data))
	if err != nil {
		return err
	}
	return ctx.Err()
}

// removeIut removes an IUT from the database.
func (s Status) removeIut(ctx context.Context, iut *checkoutable.Iut) error {
	_, err := s.db.Delete(ctx, fmt.Sprintf("%s/%s", s.iutsKey, iut.Reference))
	if err != nil {
		return err
	}
	return ctx.Err()
}

// Save saves the status struct to the database. It does not save IUTs, they are saved when added.
func (s *Status) Save(ctx context.Context) error {
	data, err := s.MarshalBinary()
	if err != nil {
		return err
	}
	for _, iut := range s.Iuts {
		s.IutReferences = append(s.IutReferences, iut.Reference)
	}
	if ctx.Err() == nil {
		_, err := s.db.Put(ctx, s.statusKey, string(data))
		if err != nil {
			return fmt.Errorf("etcd put operation failed while saving status: key=%s, value=%s", s.statusKey, string(data))
		}
	}
	return ctx.Err()
}

// Load a status struct from the data in the database.
func (s *Status) Load(ctx context.Context) error {
	response, err := s.db.Get(ctx, s.statusKey)
	if err != nil {
		return err
	}
	if len(response.Kvs) == 0 {
		return fmt.Errorf("no status found in database for %s", s.statusKey)
	}
	value := response.Kvs[0].Value
	err = json.Unmarshal(value, s)
	if err != nil {
		return err
	}

	response, err = s.db.Get(ctx, s.iutsKey, clientv3.WithPrefix())
	if err != nil {
		return err
	}
	for _, kv := range response.Kvs {
		var iut checkoutable.Iut
		err = json.Unmarshal(kv.Value, &iut)
		if err != nil {
			return err
		}
		s.Iuts = append(s.Iuts, &iut)
	}
	return ctx.Err()
}

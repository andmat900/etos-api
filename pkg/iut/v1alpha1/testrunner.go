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
	"github.com/sirupsen/logrus"
)

type Environment struct {
	IutInfo string `json:"IUT_INFO,omitempty"`
}

type Steps struct {
	Environment Environment `json:"environment"`
	Commands    []Command   `json:"commands"`
}

type TestRunner struct {
	Steps Steps `json:"steps"`
}

type Command struct {
	Name       string   `json:"name"`
	Parameters []string `json:"parameters"`
	Script     []string `json:"script"`
}

type IutInfo struct {
	Identity string `json:"identity"`
}

// AddTestRunnerInfo adds Test runner specific information to the Iut struct
func (h V1Alpha1Handler) AddTestRunnerInfo(ctx context.Context, logger *logrus.Entry, status *Status, iuts []*checkoutable.Iut) []*checkoutable.Iut {
	for idx, iut := range iuts {
		iutInfo := IutInfo{
			Identity: iut.Identity,
		}

		iutInfoJson, err := json.Marshal(iutInfo)
		if err != nil {
			err := fmt.Errorf("unable to marshall Iut Info json for %s", iut.Identity)
			if statusErr := status.Failed(ctx, err.Error()); statusErr != nil {
				logger.WithError(statusErr).Error("failed to write failure status to database")
			}
		}

		var commands []Command
		var parameters []string
		iut_info_command := Command{
			Name:       "iut_info_json",
			Parameters: append(parameters, ""),
			Script: []string{
				"#!/bin/bash",
				"mkdir -p $GLOBAL_ARTIFACT_PATH",
				"cat << EOF | python -m json.tool > $TEST_ARTIFACT_PATH/iut_info.json",
				string(iutInfoJson),
				"EOF",
			},
		}

		commands = append(commands, iut_info_command)
		//commands = append(commands, h.createDutMetadataCommand(ctx, logger, status, iuts))

		iuts[idx].TestRunner = TestRunner{
			Steps{
				Environment{
					IutInfo: string(iutInfoJson),
				},
				commands,
			},
		}

	}
	return iuts
}

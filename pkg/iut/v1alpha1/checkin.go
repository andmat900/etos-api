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
	"errors"
	"fmt"
	"time"

	"github.com/eiffel-community/etos-api/internal/iut/checkoutable"
	"github.com/sirupsen/logrus"
)

// TrimCheckin trims and returns the provided list of Iuts and also checks in the trimmed Iuts.
func (h V1Alpha1Handler) TrimCheckin(ctx context.Context, logger *logrus.Entry, iuts []*checkoutable.Iut, status *Status, size int) ([]*checkoutable.Iut, error) {
	logger.Infof("trimming the length of IUT list and checking in the removed iut(s)")

	if len(iuts) <= size {
		return nil, fmt.Errorf(
			"the trim size [%d] must be smaller than the length of the provided list [%d]",
			size,
			len(iuts),
		)
	}

	var trimmedIuts []*checkoutable.Iut
	for i := 1; i <= len(iuts)-size; i++ {
		trimmedIuts = append(trimmedIuts, iuts[len(iuts)-i])
	}

	if err := h.checkIn(ctx, logger, trimmedIuts, status, true); err != nil {
		return nil, fmt.Errorf("failed to checkin trimmed iuts - Reason: %s", err.Error())
	}

	return iuts[:size], ctx.Err()
}

// Checkin checks in the provided list of Iuts
func (h V1Alpha1Handler) checkIn(ctx context.Context, logger *logrus.Entry, iuts []*checkoutable.Iut, status *Status, withForce bool) error {
	var anyError error
	var references []string
	for _, iut := range iuts {
		var refs []string
		refs, err := iut.CheckIn(withForce)
		if err != nil {
			anyError = errors.Join(anyError, err)
			references = append(references, refs...)
			continue
		}
		references = append(references, refs...)
		failCtx, c := context.WithTimeout(context.Background(), time.Second*10)
		defer c()
		if err := status.RemoveIUT(failCtx, iut); err != nil {
			anyError = errors.Join(anyError, err)
		}
	}
	if anyError != nil {
		return errors.Join(fmt.Errorf(
			"failed to checkin Iut. Checkout refs: %+v", references,
		), anyError)
	}
	return ctx.Err()
}

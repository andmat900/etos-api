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
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/eiffel-community/etos-api/internal/iut/checkoutable"
	"github.com/google/uuid"
	"github.com/sethvargo/go-retry"
	"github.com/sirupsen/logrus"
)

func (h V1Alpha1Handler) Success(
	ctx context.Context,
	logger *logrus.Entry,
	status *Status,
	checkedOutIuts []*checkoutable.Iut,
) error {
	if err := status.Done(ctx, "IUTs checked out successfully"); err != nil {
		logger.Errorf("failed to write success status to database - Reason: %s", err.Error())
		logger.Infof("clean up, checking in any checked out IUTs")
		if checkInErr := h.checkIn(ctx, logger, checkedOutIuts, status, false); checkInErr != nil {
			logger.Errorf("clean up failure - Reason: %s", checkInErr.Error())
		}
		return err
	}
	return ctx.Err()
}

// AvailableIuts list all currently available IUTs.
func (h V1Alpha1Handler) AvailableIuts(
	ctx context.Context,
	req StartRequest,
	logger *logrus.Entry,
	identity string,
	artifactId string,
) []*checkoutable.Iut {
	availableIuts := make([]*checkoutable.Iut, 1)
	return availableIuts
}

// checkout checks out an IUT and stores it in database
func (h V1Alpha1Handler) checkout(ctx context.Context, identifier string, iut *checkoutable.Iut, req StartRequest, status *Status) error {
	if err := iut.Checkout(
		ctx,
		identifier,
		req.Context,
	); err != nil {
		return err
	}
	return status.AddIUT(ctx, iut)
}

// DoCheckout handles the checkout (and flash optionally) process for the provided iuts
func (h V1Alpha1Handler) DoCheckout(
	ctx context.Context,
	logger *logrus.Entry,
	req StartRequest,
	checkoutID uuid.UUID,
	identifier string,
) {
	h.wg.Add(1)
	defer h.wg.Done()

	// Panic recovery
	defer func() {
		if err := recover(); err != nil {
			buf := make([]byte, 2048)
			n := runtime.Stack(buf, false)
			buf = buf[:n]
			logger.Errorf("recovering from err %v\n %s", err, buf)
		}
	}()

	var err error

	ctx = h.cm.Add(ctx, identifier)
	defer h.cm.Cancel(identifier)
	response, err := h.database.Get(ctx, fmt.Sprintf("/testrun/%s/provider/iut", identifier))
	if err != nil {
		msg := fmt.Errorf("error getting testrun from database - %s", err.Error())
		logger.Error(msg)
		return
	}
	if len(response.Kvs) == 0 {
		msg := fmt.Errorf("no testrun available for checkout with id: %s", identifier)
		logger.Error(msg)
		return
	}

	key := fmt.Sprintf("/testrun/%s", identifier)
	logger.Debugf("storing IUT information at %s", key)
	status := NewStatus(key, checkoutID.String(), h.database)

	greed := 0.8

	ctx, cancel := context.WithTimeout(ctx, h.cfg.IutWaitTimeoutHard())
	defer cancel()

	if err = status.Pending(ctx, "checking out IUTs"); err != nil {
		msg := fmt.Errorf("failed to write checkout pending status to database - %s", err.Error())
		logger.Error(msg)
		return
	}

	var successIuts []*checkoutable.Iut
	var iuts []*checkoutable.Iut
	// Due to a how retry-go implements WithCappedDuration, we need to have a
	// base duration that is larger than the jitter. This is because if a
	// negative jitter value causes the duration to become negative the
	// WithCappedDuration function handles this by returning the cap duration.
	backOff := retry.WithCappedDuration(
		60*time.Second,
		retry.WithJitter(
			2*time.Second,
			retry.NewFibonacci(
				5*time.Second,
			),
		),
	)
	// Try to checkout all the "available" Iuts and assess if we got enough to return successfully,
	// if not we retry until we do or the context timeout (default 1h) kicks in.
	if err := retry.Do(ctx, backOff, func(ctx context.Context) error {
		logger.Debugf("length before update %d", len(iuts))
		iuts = h.AvailableIuts(ctx, req, logger, req.ArtifactIdentity, req.ArtifactID)
		logger.Debugf("length after update %d", len(iuts))
		if len(iuts) < req.MinimumAmount {
			return fmt.Errorf(
				"Not enough Iuts available. Number of available iuts are: %d",
				len(iuts),
			)
		}

		fairAmount := fair(len(iuts), greed)
		if fairAmount >= req.MaximumAmount {
			fairAmount = req.MaximumAmount
		}
		success := make(chan *checkoutable.Iut, len(iuts))
		wg := sync.WaitGroup{}

		startTimestamp := time.Now().Unix()

		for _, iut := range iuts {
			wg.Add(1)
			// Goroutine to checkout an Iut, and to notify which were successful or not.
			go func(ctx context.Context, iut *checkoutable.Iut, req StartRequest) {
				defer wg.Done()
				select {
				case <-ctx.Done():
					return
				default:
					if err := h.checkout(ctx, identifier, iut, req, status); err != nil {
						iut.CheckIn(true)
						logger.Error(err)
						return
					}
					success <- iut
				}
			}(ctx, iut, req)
			time.Sleep(1 * time.Second)
		}
		wg.Wait()
		duration := time.Now().Unix() - startTimestamp
		softTimeout := int64(h.cfg.IutWaitTimeoutSoft().Seconds())

		if duration > softTimeout {
			logger.Infof("IUT checkout time exceeded soft timeout %d seconds", softTimeout)
		}

		close(success)
		for iut := range success {
			successIuts = append(successIuts, iut)
		}
		logger.Infof("all checkout attempts done, %d IUTs successfully checked out", len(successIuts))

		if len(successIuts) < req.MinimumAmount {
			logger.Infof("Not enough Iuts were checked out, length is: %d", len(successIuts))
			logger.WithField("user_log", true).Info("Not enough IUTs available yet")
			if err := h.checkIn(ctx, logger, successIuts, status, true); err != nil {
				return err
			}
			successIuts = nil
			return retry.RetryableError(fmt.Errorf("retrying... "))
		}

		if len(successIuts) > fairAmount {
			successIuts, err = h.TrimCheckin(ctx, logger, successIuts, status, fairAmount)
			if err != nil {
				msg := fmt.Errorf("Error occured during trim check-in (fair amount: %d): %s", fairAmount, err.Error())
				logger.Error(msg)
			}
		}
		return nil
	}); err != nil {
		if ctx.Err() != nil {
			err = fmt.Errorf("Timed out checking out IUT")
			logger.WithField("user_log", true).Error(err.Error())
		} else {
			logger.WithField("user_log", true).Errorf("Failed checking out IUT(s) - %s", err.Error())
		}
		failCtx, c := context.WithTimeout(context.Background(), time.Second*10)
		defer c()
		if statusErr := status.Failed(failCtx, err.Error()); statusErr != nil {
			logger.WithError(statusErr).Error("failed to write failure status to database")
		}
		return
	}
	successIuts = h.AddTestRunnerInfo(ctx, logger, status, successIuts)
	for _, iut := range successIuts {
		if err := status.AddIUT(ctx, iut); err != nil {
			//msg := fmt.Errorf("Failed to add IUT to database: %s - Reason: %s", iut, err.Error())
			if statusErr := status.Failed(ctx, err.Error()); statusErr != nil {
				logger.WithError(statusErr).Error("failed to write failure status to database: ")
			}
		}
	}

	logger.WithField("user_log", true).Infof("Successfully checked out %d IUTs", len(successIuts))
	for _, iut := range successIuts {
		logger.WithField("user_log", true).Infof("Reference: %s", iut.Reference)
	}
}

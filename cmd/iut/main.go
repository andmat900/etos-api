// Copyright 2022 Axis Communications AB.
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
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	config "github.com/eiffel-community/etos-api/internal/configs/iut"
	"github.com/eiffel-community/etos-api/internal/iut/contextmanager"
	server "github.com/eiffel-community/etos-api/internal/iut/server"
	"github.com/eiffel-community/etos-api/internal/logging"
	"github.com/eiffel-community/etos-api/pkg/iut/application"
	"github.com/eiffel-community/etos-api/pkg/iut/v1alpha1"
	"github.com/sirupsen/logrus"
	"github.com/snowzach/rotatefilehook"
	"go.elastic.co/ecslogrus"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// main sets up logging and starts up the webserver.
func main() {
	cfg := config.Get()
	ctx := context.Background()

	var hooks []logrus.Hook
	if fileHook := fileLogging(cfg); fileHook != nil {
		hooks = append(hooks, fileHook)
	}

	logger, err := logging.Setup(cfg.LogLevel(), hooks)
	if err != nil {
		logrus.Fatal(err.Error())
	}

	hostname, err := os.Hostname()
	if err != nil {
		logrus.Fatal(err.Error())
	}
	log := logger.WithFields(logrus.Fields{
		"hostname":    hostname,
		"application": "Dummy IUT Provider Service",
		"version":     vcsRevision(),
		"name":        "Dummy IUT Provider",
		"user_log":    false,
	})

	// Database connection test
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{cfg.DatabaseURI()},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.WithError(err).Fatal("failed to create etcd connection")
	}

	cm := contextmanager.New(cli)
	go cm.Start(ctx)
	defer cm.CancelAll()

	log.Info("Loading v1alpha1 routes")
	v1alpha1App := v1alpha1.New(cfg, log, ctx, cm, cli)
	defer v1alpha1App.Close()
	router := application.New(v1alpha1App)

	srv := server.NewWebserver(cfg, log, router)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Errorf("Webserver shutdown: %+v", err)
		}
	}()

	<-done
	log.Info("SIGTERM received")

	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout())
	defer cancel()
	v1alpha1App.Close()

	if err := srv.Close(ctx); err != nil {
		log.Errorf("Webserver shutdown failed: %+v", err)
	}
	log.Info("Wait for checkout, deploy, status and checkin jobs to complete")
}

// fileLogging adds a hook into a slice of hooks, if the filepath configuration is set
func fileLogging(cfg config.Config) logrus.Hook {
	if filePath := cfg.LogFilePath(); filePath != "" {
		// TODO: Make these parameters configurable.
		// NewRotateFileHook cannot return an error which is why it's set to '_'.
		rotateFileHook, _ := rotatefilehook.NewRotateFileHook(rotatefilehook.RotateFileConfig{
			Filename:   filePath,
			MaxSize:    10, // megabytes
			MaxBackups: 3,
			MaxAge:     0, // days
			Level:      logrus.DebugLevel,
			Formatter: &ecslogrus.Formatter{
				DataKey: "labels",
			},
		})
		return rotateFileHook
	}
	return nil
}

func vcsRevision() string {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return "(unknown)"
	}
	for _, val := range buildInfo.Settings {
		if val.Key == "vcs.revision" {
			return val.Value
		}
	}
	return "(unknown)"
}

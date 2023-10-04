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
package kubernetes

import (
	"context"
	"fmt"

	"github.com/eiffel-community/etos-api/internal/config"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Kubernetes struct {
	logger    *logrus.Entry
	config    *rest.Config
	client    *kubernetes.Clientset
	namespace string
}

// New creates a new Kubernetes struct.
func New(cfg config.Config, log *logrus.Entry) *Kubernetes {
	return &Kubernetes{
		logger:    log,
		namespace: cfg.ETOSNamespace(),
	}
}

// kubeconfig gets a kubeconfig file.
func (k *Kubernetes) kubeconfig() (*rest.Config, error) {
	return rest.InClusterConfig()
}

// clientset creates a new Kubernetes client
func (k *Kubernetes) clientset() (*kubernetes.Clientset, error) {
	if k.client != nil {
		return k.client, nil
	}
	if k.config == nil {
		cfg, err := k.kubeconfig()
		if err != nil {
			return nil, err
		}
		k.config = cfg
	}
	cli, err := kubernetes.NewForConfig(k.config)
	if err != nil {
		return nil, err
	}
	k.client = cli
	return k.client, nil
}

// IsFinished checks if an ESR job is finished.
func (k *Kubernetes) IsFinished(ctx context.Context, identifier string) bool {
	client, err := k.clientset()
	if err != nil {
		k.logger.Error(err)
		return false
	}
	jobs, err := client.BatchV1().Jobs(k.namespace).List(
		ctx,
		metav1.ListOptions{
			LabelSelector: fmt.Sprintf("id=%s", identifier),
		},
	)
	if err != nil {
		k.logger.Error(err)
		return false
	}
	if len(jobs.Items) == 0 {
		// Assume that a job is finished if it does not exist.
		k.logger.Warningf("job with id %s does not exist, assuming finished", identifier)
		return true
	}
	job := jobs.Items[0]
	if job.Status.Succeeded == 0 && job.Status.Failed == 0 {
		return false
	}
	return true
}

// LogListenerIP gets the IP address of an ESR log listener.
func (k *Kubernetes) LogListenerIP(ctx context.Context, identifier string) (string, error) {
	client, err := k.clientset()
	if err != nil {
		return "", err
	}
	jobs, err := client.BatchV1().Jobs(k.namespace).List(
		ctx,
		metav1.ListOptions{
			LabelSelector: fmt.Sprintf("id=%s", identifier),
		},
	)
	if err != nil {
		return "", err
	}
	if len(jobs.Items) == 0 {
		return "", fmt.Errorf("could not find esr job with id %s", identifier)
	}
	job := jobs.Items[0]

	pods, err := client.CoreV1().Pods(k.namespace).List(
		ctx,
		metav1.ListOptions{
			LabelSelector: fmt.Sprintf("job-name=%s", job.Name),
		},
	)
	if err != nil {
		return "", err
	}
	if len(pods.Items) == 0 {
		return "", fmt.Errorf("could not find pod for job with id %s", identifier)
	}
	pod := pods.Items[0]
	return pod.Status.PodIP, nil
}

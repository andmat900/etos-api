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
package contextmanager

import (
	"context"
	"regexp"
	"strings"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const REGEX = "/testrun/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/provider/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/status"

// ContextManager manages contexts for the IUT provider goroutines, enabling the cancelation
// of the context in the case where a testrun should be removed during the checkout phase.
type ContextManager struct {
	contexts map[string]context.CancelFunc
	db       *clientv3.Client
}

// New creates a new context manager.
func New(db *clientv3.Client) *ContextManager {
	return &ContextManager{
		contexts: make(map[string]context.CancelFunc),
		db:       db,
	}
}

// CancelAll cancels all saved contexts within the context manager.
func (c *ContextManager) CancelAll() {
	for _, cancel := range c.contexts {
		cancel()
	}
	c.contexts = make(map[string]context.CancelFunc)
}

// Start up the context manager testrun deletaion watcher.
func (c *ContextManager) Start(ctx context.Context) {
	regex := regexp.MustCompile(REGEX)
	ch := c.db.Watch(ctx, "/testrun", clientv3.WithPrefix())
	for response := range ch {
		for _, event := range response.Events {
			if event.Type == mvccpb.DELETE {
				if !regex.Match(event.Kv.Key) {
					continue
				}
				dbPath := strings.Split(string(event.Kv.Key), "/")
				c.Cancel(dbPath[2])
			}
		}
	}
}

// Cancel the context for one stored ID.
func (c *ContextManager) Cancel(id string) {
	cancel, ok := c.contexts[id]
	if !ok {
		return
	}
	delete(c.contexts, id)
	cancel()
}

// Add a new context to the context manager.
func (c *ContextManager) Add(ctx context.Context, id string) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	c.contexts[id] = cancel
	return ctx
}

// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package apisix

import (
	"context"
	"net"
	"testing"
	"time"

	adapter "github.com/api7/etcd-adapter/pkg/adapter"
	"github.com/api7/etcd-adapter/pkg/backends/btree"
	"github.com/stretchr/testify/require"
)

func TestEtcdCreateResourceReconcilesExistingConfiguration(t *testing.T) {
	for _, tc := range []struct {
		resource string
		initial  string
		ready    string
	}{
		{"upstreams", `{"id":"console","type":"roundrobin","nodes":{}}`, `{"id":"console","type":"roundrobin","nodes":{"10.42.0.7:7070":1}}`},
		{"routes", `{"id":"console","uri":"/*","upstream_id":"initial"}`, `{"id":"console","uri":"/*","upstream_id":"ready"}`},
		{"stream_routes", `{"id":"console","server_port":7070,"upstream_id":"initial"}`, `{"id":"console","server_port":7070,"upstream_id":"ready"}`},
	} {
		t.Run(tc.resource, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			backend := btree.NewBTreeCache()
			storage := adapter.NewEtcdAdapter(&adapter.AdapterOptions{Backend: backend})
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			go func() { _ = storage.Serve(ctx, listener) }()
			t.Cleanup(func() {
				shutdownCtx, stop := context.WithTimeout(context.Background(), time.Second)
				defer stop()
				_ = storage.Shutdown(shutdownCtx)
				cancel()
				_ = listener.Close()
			})

			cluster := &cluster{adapter: storage, prefix: "/apisix"}
			key := "/apisix/" + tc.resource + "/console"
			storedValue := func() string {
				_, value, getErr := backend.Get(ctx, key, "0", 0, 0)
				if getErr != nil || value == nil {
					return ""
				}
				return string(value.Value)
			}
			cluster.CreateResource(tc.resource, "console", []byte(tc.initial))
			require.Eventually(t, func() bool { return storedValue() == tc.initial }, time.Second, 10*time.Millisecond)

			// A repeated reconciliation must replace the initial empty upstream,
			// rather than leave APISIX serving its stale startup configuration.
			cluster.CreateResource(tc.resource, "console", []byte(tc.ready))
			require.Eventually(t, func() bool { return storedValue() == tc.ready }, time.Second, 10*time.Millisecond)

			cluster.DeleteResource(tc.resource, "console", nil)
			require.Eventually(t, func() bool { return storedValue() == "" }, time.Second, 10*time.Millisecond)
			cluster.CreateResource(tc.resource, "console", []byte(tc.ready))
			require.Eventually(t, func() bool { return storedValue() == tc.ready }, time.Second, 10*time.Millisecond)
		})
	}
}

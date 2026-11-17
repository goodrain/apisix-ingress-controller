// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package endpoint

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/workqueue"

	"github.com/apache/apisix-ingress-controller/pkg/metrics"
	"github.com/apache/apisix-ingress-controller/pkg/providers/k8s/namespace"
	providertypes "github.com/apache/apisix-ingress-controller/pkg/providers/types"
)

func endpoints(resourceVersion string) *corev1.Endpoints {
	return &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       "default",
			Name:            "console",
			ResourceVersion: resourceVersion,
		},
	}
}

// Resource versions are opaque strings. On a fresh cluster they grow from
// three to four digits, and a lexical comparison ("939" >= "1625") used to
// drop the update that made the backend ready, leaving the upstream empty.
func TestEndpointsOnUpdateKeepsNewerVersionsOfAnyLength(t *testing.T) {
	cases := []struct {
		prev, curr string
		queued     int
	}{
		{prev: "939", curr: "1625", queued: 1},
		{prev: "1135", curr: "1591", queued: 1},
		{prev: "1625", curr: "1625", queued: 0},
	}
	for _, tc := range cases {
		ctl := &endpointsController{
			baseEndpointController: &baseEndpointController{
				Common: &providertypes.Common{MetricsCollector: metrics.NewPrometheusCollector()},
			},
			workqueue:         workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
			namespaceProvider: namespace.NewMockWatchingNamespaceProvider([]string{"default"}),
		}
		ctl.onUpdate(endpoints(tc.prev), endpoints(tc.curr))
		assert.Equal(t, tc.queued, ctl.workqueue.Len(), "prev=%s curr=%s", tc.prev, tc.curr)
		ctl.workqueue.ShutDown()
	}
}

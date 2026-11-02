// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package v2

import (
	"reflect"
	"sync"
	"testing"
)

func pluginConfigFixture() map[string]interface{} {
	return map[string]interface{}{
		"number": 1,
		"nested": map[string]interface{}{"value": "original"},
		"list":   []interface{}{map[string]interface{}{"value": "original"}},
	}
}

func TestPluginConfigDeepCopyIntoIndependent(t *testing.T) {
	for name, copyInto := range map[string]func(map[string]interface{}, map[string]interface{}) map[string]interface{}{
		"route": func(in, out map[string]interface{}) map[string]interface{} {
			src, dst := ApisixRoutePluginConfig(in), ApisixRoutePluginConfig(out)
			src.DeepCopyInto(&dst)
			return dst
		},
		"global": func(in, out map[string]interface{}) map[string]interface{} {
			src, dst := Config(in), Config(out)
			src.DeepCopyInto(&dst)
			return dst
		},
	} {
		t.Run(name, func(t *testing.T) {
			src := pluginConfigFixture()
			dst := copyInto(src, src)
			if !reflect.DeepEqual(src, pluginConfigFixture()) {
				t.Fatal("copying changed the source configuration")
			}
			dst["added"] = true
			dst["nested"].(map[string]interface{})["value"] = "changed"
			dst["list"].([]interface{})[0].(map[string]interface{})["value"] = "changed"
			if !reflect.DeepEqual(src, pluginConfigFixture()) {
				t.Fatal("copy still shares a map or slice with its source")
			}
			for _, input := range []map[string]interface{}{nil, {}} {
				out := copyInto(input, map[string]interface{}{"stale": true})
				if !reflect.DeepEqual(out, input) {
					t.Fatalf("copy must replace its destination and preserve nil/empty: got %#v, want %#v", out, input)
				}
			}
		})
	}
}

func TestConfigSetDeepCopyIntoIndependent(t *testing.T) {
	src := ConfigSet{pluginConfigFixture()}
	dst := src
	src.DeepCopyInto(&dst)
	if !reflect.DeepEqual(src, ConfigSet{pluginConfigFixture()}) {
		t.Fatal("copying changed the source configuration set")
	}
	dst[0].(map[string]interface{})["nested"].(map[string]interface{})["value"] = "changed"
	dst[0] = nil
	if !reflect.DeepEqual(src, ConfigSet{pluginConfigFixture()}) {
		t.Fatal("copy still shares its backing array or nested configuration")
	}
	for _, input := range []ConfigSet{nil, {}} {
		out := ConfigSet{true}
		input.DeepCopyInto(&out)
		if !reflect.DeepEqual(out, input) {
			t.Fatalf("copy must preserve nil/empty: got %#v, want %#v", out, input)
		}
	}
}

func TestApisixRouteDeepCopyObjectIndependent(t *testing.T) {
	src := &ApisixRoute{Spec: ApisixRouteSpec{HTTP: []ApisixRouteHTTP{{
		Plugins: []ApisixRoutePlugin{{Name: "proxy-rewrite", Config: pluginConfigFixture()}},
	}}}}
	dst := src.DeepCopyObject().(*ApisixRoute)
	dst.Spec.HTTP[0].Plugins[0].Config["added"] = true
	if !reflect.DeepEqual(map[string]interface{}(src.Spec.HTTP[0].Plugins[0].Config), pluginConfigFixture()) {
		t.Fatal("generated route copy changed the informer object")
	}
}

func TestApisixRouteDeepCopyObjectConcurrent(t *testing.T) {
	src := &ApisixRoute{Spec: ApisixRouteSpec{HTTP: []ApisixRouteHTTP{{
		Plugins: []ApisixRoutePlugin{{Name: "proxy-rewrite", Config: pluginConfigFixture()}},
	}}}}
	var workers sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 8; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			for j := 0; j < 100; j++ {
				dst := src.DeepCopyObject().(*ApisixRoute)
				dst.Spec.HTTP[0].Plugins[0].Config["worker"] = j
			}
		}()
	}
	close(start)
	workers.Wait()
	if !reflect.DeepEqual(map[string]interface{}(src.Spec.HTTP[0].Plugins[0].Config), pluginConfigFixture()) {
		t.Fatal("concurrent route copies changed the informer object")
	}
}

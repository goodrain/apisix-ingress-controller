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
package v1

import (
	"reflect"
	"sync"
	"testing"
)

func pluginsFixture() Plugins {
	return Plugins{"proxy-rewrite": map[string]interface{}{
		"headers":   map[string]interface{}{"X-Test": "original"},
		"regex_uri": []interface{}{"^/api/(.*)", "/$1"},
	}}
}

func TestRouteDeepCopyPluginsIndependent(t *testing.T) {
	src := &Route{Plugins: pluginsFixture()}
	dst := src.DeepCopy()
	dst.Plugins["added"] = true
	config := dst.Plugins["proxy-rewrite"].(map[string]interface{})
	config["headers"].(map[string]interface{})["X-Test"] = "changed"
	config["regex_uri"].([]interface{})[0] = "changed"
	if !reflect.DeepEqual(src.Plugins, pluginsFixture()) {
		t.Fatal("generated route copy still shares plugin configuration with its source")
	}
}

func TestPluginsDeepCopyIntoReplacesDestination(t *testing.T) {
	for _, input := range []Plugins{nil, {}, pluginsFixture()} {
		out := Plugins{"stale": true}
		input.DeepCopyInto(&out)
		if !reflect.DeepEqual(out, input) {
			t.Fatalf("copy must replace its destination and preserve nil/empty: got %#v, want %#v", out, input)
		}
	}
}

func TestRewriteConfigDeepCopyHeadersIndependent(t *testing.T) {
	src := &RewriteConfig{Headers: Headers{"set": map[string]interface{}{"X-Test": "original"}}}
	dst := src.DeepCopy()
	dst.Headers["added"] = true
	dst.Headers["set"].(map[string]interface{})["X-Test"] = "changed"
	if !reflect.DeepEqual(src.Headers, Headers{"set": map[string]interface{}{"X-Test": "original"}}) {
		t.Fatal("generated rewrite copy still shares headers with its source")
	}
	for _, input := range []Headers{nil, {}} {
		out := Headers{"stale": true}
		input.DeepCopyInto(&out)
		if !reflect.DeepEqual(out, input) {
			t.Fatalf("copy must replace its destination and preserve nil/empty: got %#v, want %#v", out, input)
		}
	}
}

func TestRouteDeepCopyPluginsConcurrent(t *testing.T) {
	src := &Route{Plugins: pluginsFixture()}
	var workers sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 8; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			for j := 0; j < 100; j++ {
				dst := src.DeepCopy()
				dst.Plugins["worker"] = j
			}
		}()
	}
	close(start)
	workers.Wait()
	if !reflect.DeepEqual(src.Plugins, pluginsFixture()) {
		t.Fatal("concurrent route copies changed their source")
	}
}

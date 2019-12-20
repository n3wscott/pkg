// +build gke

/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package actions

import (
	"knative.dev/pkg/testutils/clustermanager/prow-cluster-operation/options"
)

// Get gets a GKE cluster
func Get(o *options.RequestWrapper) error {
	o.Prep()
	o.Request.SkipCreation = true
	// Reuse `Create` for getting operation, so that we can reuse the same logic
	// such as protected project/cluster etc.
	return Create(o)
}

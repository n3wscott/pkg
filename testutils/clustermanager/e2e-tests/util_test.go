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

package clustermanager

import (
	"testing"

	"knative.dev/pkg/testutils/clustermanager/e2e-tests/common"
)

func TestGetResourceName(t *testing.T) {
	datas := []struct {
		isProw      bool
		buildNumStr string
		exp         string
	}{
		{true, "12345678901234567890fakebuildnum", "kpkg-e2e-cls-12345678901234567890"},
		{false, "", "kpkg-e2e-cls"},
	}

	// mock GetOSEnv for testing
	oldFunc := common.GetOSEnv
	defer func() {
		// restore GetOSEnv
		common.GetOSEnv = oldFunc
	}()
	for _, data := range datas {
		common.GetOSEnv = func(key string) string {
			if data.isProw {
				switch key {
				case "BUILD_NUMBER":
					return data.buildNumStr
				case "PROW_JOB_ID": // needed to mock IsProw()
					return "jobid"
				}
			}
			return ""
		}

		out, err := getResourceName(ClusterResource)
		if err != nil {
			t.Fatalf("getting resource name for cluster, want: 'no error', got: '%v'", err)
		}
		if out != data.exp {
			t.Fatalf("getting resource name for cluster, want: '%s', got: '%s'", data.exp, out)
		}
	}
}

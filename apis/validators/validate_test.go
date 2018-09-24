/*
Copyright 2018 The Knative Authors

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

package validators

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/knative/pkg/apis"
)

type foo struct {
	Default string `validate:"-"`
}

type foo_k8s struct {
	Default      string `json:"default,omitempty"`
	OptionalName string `json:"optionalName" validate:"QualifiedName"`
	RequiredName string `json:"requiredName" validate:"QualifiedName,Required"`
}

type non_json_k8s struct {
	OptionalName string `validate:"QualifiedName"`
	RequiredName string `validate:"QualifiedName,Required"`
}

const invalidQualifiedNameError = `name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')`

func TestValidate(t *testing.T) {
	type args struct {
		obj interface{}
	}
	tests := []struct {
		name string
		args args
		want *apis.FieldError
	}{{
		name: "default",
		args: args{
			obj: foo{
				Default: "default",
			},
		},
		want: nil,
	}, {
		name: "valid k8s",
		args: args{
			obj: foo_k8s{
				Default:      "default",
				OptionalName: "valid",
				RequiredName: "valid",
			},
		},
		want: nil,
	}, {
		name: "missing required k8s name",
		args: args{
			obj: foo_k8s{},
		},
		want: &apis.FieldError{
			Message: `missing field(s)`,
			Paths:   []string{"requiredName"},
		},
	}, {
		name: "invalid optional k8s name",
		args: args{
			obj: foo_k8s{
				Default:      "default",
				OptionalName: "v@lid",
				RequiredName: "valid",
			},
		},
		want: &apis.FieldError{
			Message: `invalid key name "v@lid"`,
			Paths:   []string{"optionalName"},
			Details: invalidQualifiedNameError,
		},
	}, {
		name: "invalid required k8s name",
		args: args{
			obj: foo_k8s{
				RequiredName: "v@lid",
			},
		},
		want: &apis.FieldError{
			Message: `invalid key name "v@lid"`,
			Paths:   []string{"requiredName"},
			Details: invalidQualifiedNameError,
		},
	}, {
		name: "invalid optional and required k8s names",
		args: args{
			obj: foo_k8s{
				OptionalName: "val!d",
				RequiredName: "v@lid",
			},
		},
		want: (&apis.FieldError{
			Message: `invalid key name "val!d"`,
			Paths:   []string{"optionalName"},
			Details: invalidQualifiedNameError,
		}).Also(&apis.FieldError{
			Message: `invalid key name "v@lid"`,
			Paths:   []string{"requiredName"},
			Details: invalidQualifiedNameError,
		}),
	}, {
		name: "non-json invalid optional and required k8s names",
		args: args{
			obj: non_json_k8s{
				OptionalName: "val!d",
				RequiredName: "v@lid",
			},
		},
		want: (&apis.FieldError{
			Message: `invalid key name "val!d"`,
			Paths:   []string{"OptionalName"},
			Details: invalidQualifiedNameError,
		}).Also(&apis.FieldError{
			Message: `invalid key name "v@lid"`,
			Paths:   []string{"RequiredName"},
			Details: invalidQualifiedNameError,
		}),
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Validate(tt.args.obj)
			if diff := cmp.Diff(tt.want.Error(), got.Error()); diff != "" {
				t.Errorf("Validate() (-want, +got) = %v", diff)
			}
		})
	}
}
/*
Copyright 2017 The Knative Authors

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

package apis

import (
	"strconv"
	"strings"
	"testing"
)

func TestFieldError(t *testing.T) {
	tests := []struct {
		name     string
		err      *FieldError
		prefixes [][]string
		want     string
	}{{
		name: "simple single no propagation",
		err: &FieldError{
			Message: "hear me roar",
			Paths:   []string{"foo.bar"},
		},
		want: "hear me roar: foo.bar",
	}, {
		name: "simple single propagation",
		err: &FieldError{
			Message: `invalid value "blah"`,
			Paths:   []string{"foo"},
		},
		prefixes: [][]string{{"bar"}, {"baz", "ugh"}, {"hoola"}},
		want:     `invalid value "blah": hoola.baz.ugh.bar.foo`,
	}, {
		name: "simple multiple propagation",
		err: &FieldError{
			Message: "invalid field(s)",
			Paths:   []string{"foo", "bar"},
		},
		prefixes: [][]string{{"baz", "ugh"}},
		want:     "invalid field(s): baz.ugh.bar, baz.ugh.foo",
	}, {
		name: "multiple propagation with details",
		err: &FieldError{
			Message: "invalid field(s)",
			Paths:   []string{"foo", "bar"},
			Details: `I am a long
long
loooong
Body.`,
		},
		prefixes: [][]string{{"baz", "ugh"}},
		want: `invalid field(s): baz.ugh.bar, baz.ugh.foo
I am a long
long
loooong
Body.`,
	}, {
		name: "single propagation, empty start",
		err: &FieldError{
			Message: "invalid field(s)",
			// We might see this validating a scalar leaf.
			Paths: []string{CurrentField},
		},
		prefixes: [][]string{{"baz", "ugh"}},
		want:     "invalid field(s): baz.ugh",
	}, {
		name: "single propagation, no paths",
		err: &FieldError{
			Message: "invalid field(s)",
			Paths:   nil,
		},
		prefixes: [][]string{{"baz", "ugh"}},
		want:     "invalid field(s): ",
	}, {
		name:     "nil propagation",
		err:      nil,
		prefixes: [][]string{{"baz", "ugh"}},
	}, {
		name:     "missing field propagation",
		err:      ErrMissingField("foo", "bar"),
		prefixes: [][]string{{"baz"}},
		want:     "missing field(s): baz.bar, baz.foo",
	}, {
		name:     "missing disallowed propagation",
		err:      ErrDisallowedFields("foo", "bar"),
		prefixes: [][]string{{"baz"}},
		want:     "must not set the field(s): baz.bar, baz.foo",
	}, {
		name:     "invalid value propagation",
		err:      ErrInvalidValue("foo", "bar"),
		prefixes: [][]string{{"baz"}},
		want:     `invalid value "foo": baz.bar`,
	}, {
		name:     "missing mutually exclusive fields",
		err:      ErrMissingOneOf("foo", "bar"),
		prefixes: [][]string{{"baz"}},
		want:     `expected exactly one, got neither: baz.bar, baz.foo`,
	}, {
		name:     "multiple mutually exclusive fields",
		err:      ErrMultipleOneOf("foo", "bar"),
		prefixes: [][]string{{"baz"}},
		want:     `expected exactly one, got both: baz.bar, baz.foo`,
	}, {
		name: "invalid key name",
		err: ErrInvalidKeyName("b@r", "foo[0].name",
			"can not use @", "do not try"),
		prefixes: [][]string{{"baz"}},
		want: `invalid key name "b@r": baz.foo[0].name
can not use @, do not try`,
	}, {
		name: "invalid key name with details array",
		err: ErrInvalidKeyName("b@r", "foo[0].name",
			[]string{"can not use @", "do not try"}...),
		prefixes: [][]string{{"baz"}},
		want: `invalid key name "b@r": baz.foo[0].name
can not use @, do not try`,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fe := test.err
			// Simulate propagation up a call stack.
			for _, prefix := range test.prefixes {
				fe = fe.ViaField(prefix...)
			}
			if test.want != "" {
				got := fe.Error()
				if got != test.want {
					t.Errorf("%s: Error() = %v, wanted %v", test.name, got, test.want)
				}
			} else if fe != nil {
				t.Errorf("%s: ViaField() = %v, wanted nil", test.name, fe)
			}
		})
	}
}

func TestViaIndexOrKeyFieldError(t *testing.T) {
	tests := []struct {
		name     string
		err      *FieldError
		prefixes [][]string
		want     string
	}{{
		name: "nil",
		err:  nil,
		want: "",
	}, {
		name:     "nil with prefix",
		err:      nil,
		prefixes: [][]string{{"INDEX:2"}, {"KEY:B"}, {"FIELDINDEX:6,AAA"}, {"FIELDKEY:bee,AAA"}},
		want:     "",
	}, {
		name: "simple single no propagation",
		err: &FieldError{
			Message: "hear me roar",
			Paths:   []string{"bar"},
		},
		prefixes: [][]string{{"INDEX:3", "INDEX:2", "INDEX:1", "foo"}},
		want:     "hear me roar: foo[1][2][3].bar",
	}, {
		name: "simple key",
		err: &FieldError{
			Message: "hear me roar",
			Paths:   []string{"bar"},
		},
		prefixes: [][]string{{"KEY:C", "KEY:B", "KEY:A", "foo"}},
		want:     "hear me roar: foo[A][B][C].bar",
	}, {
		name:     "missing field propagation",
		err:      ErrMissingField("foo", "bar"),
		prefixes: [][]string{{"[2]", "baz"}},
		want:     "missing field(s): baz[2].bar, baz[2].foo",
	}, {
		name: "invalid key name",
		err: ErrInvalidKeyName("b@r", "name",
			"can not use @", "do not try"),
		prefixes: [][]string{{"baz", "INDEX:0", "foo"}},
		want: `invalid key name "b@r": foo[0].baz.name
can not use @, do not try`,
	}, {
		name: "invalid key name with keys",
		err: ErrInvalidKeyName("b@r", "name",
			"can not use @", "do not try"),
		prefixes: [][]string{{"baz", "INDEX:0", "foo"}, {"bar", "KEY:A", "boo"}},
		want: `invalid key name "b@r": boo[A].bar.foo[0].baz.name
can not use @, do not try`,
	}, {
		name: "multi prefixes provided",
		err: &FieldError{
			Message: "invalid field(s)",
			Paths:   []string{"foo"},
		},
		prefixes: [][]string{{"INDEX:2"}, {"bee"}, {"INDEX:0"}, {"baa", "baz", "ugh"}},
		want:     "invalid field(s): ugh.baz.baa[0].bee[2].foo",
	}, {
		name: "use helper viaFieldIndex",
		err: &FieldError{
			Message: "invalid field(s)",
			Paths:   []string{"foo"},
		},
		prefixes: [][]string{{"FIELDINDEX:bee,2"}, {"FIELDINDEX:baa,0"}, {"baz", "ugh"}},
		want:     "invalid field(s): ugh.baz.baa[0].bee[2].foo",
	}, {
		name: "use helper viaFieldKey",
		err: &FieldError{
			Message: "invalid field(s)",
			Paths:   []string{"foo"},
		},
		prefixes: [][]string{{"FIELDKEY:bee,AAA"}, {"FIELDKEY:baa,BBB"}, {"baz", "ugh"}},
		want:     "invalid field(s): ugh.baz.baa[BBB].bee[AAA].foo",
	}, {
		name: "bypass helpers",
		err: &FieldError{
			Message: "invalid field(s)",
			Paths:   []string{"foo"},
		},
		prefixes: [][]string{{"[2]"}, {"[1]"}, {"bar"}},
		want:     "invalid field(s): bar[1][2].foo",
	}, {
		name: "multi paths provided",
		err: &FieldError{
			Message: "invalid field(s)",
			Paths:   []string{"foo", "bar"},
		},
		prefixes: [][]string{{"INDEX:0"}, {"index"}, {"KEY:A"}, {"map"}},
		want:     "invalid field(s): map[A].index[0].bar, map[A].index[0].foo",
	}, {
		name: "manual index",
		err: func() *FieldError {
			// Example, return an error in a loop:
			// for i, item := spec.myList {
			//   err := item.validate().ViaIndex(i).ViaField("myList")
			//   if err != nil {
			// 		return err
			//   }
			// }
			// --> I expect path to be myList[i].foo

			err := &FieldError{
				Message: "invalid field(s)",
				Paths:   []string{"foo"},
			}

			err = err.ViaIndex(0).ViaField("bar")
			err = err.ViaIndex(2).ViaIndex(1).ViaField("baz")
			err = err.ViaIndex(3).ViaIndex(4).ViaField("boof")
			return err
		}(),
		want: "invalid field(s): boof[4][3].baz[1][2].bar[0].foo",
	}, {
		name: "manual multiple index",
		err: func() *FieldError {

			err := &FieldError{
				Message: "invalid field(s)",
				Paths:   []string{"foo"},
			}

			err = err.ViaField("bear", "[1]", "[2]", "[3]", "baz", "]xxx[").ViaField("bar")
			return err
		}(),
		want: "invalid field(s): bar.bear[1][2][3].baz.]xxx[.foo",
	}, {
		name: "manual keys",
		err: func() *FieldError {
			err := &FieldError{
				Message: "invalid field(s)",
				Paths:   []string{"foo"},
			}

			err = err.ViaKey("A").ViaField("bar")
			err = err.ViaKey("CCC").ViaKey("BB").ViaField("baz")
			err = err.ViaKey("E").ViaKey("F").ViaField("jar")
			return err
		}(),
		want: "invalid field(s): jar[F][E].baz[BB][CCC].bar[A].foo",
	}, {
		name: "manual index and keys",
		err: func() *FieldError {
			err := &FieldError{
				Message: "invalid field(s)",
				Paths:   []string{"foo", "faa"},
			}
			err = err.ViaKey("A").ViaField("bar")
			err = err.ViaIndex(1).ViaField("baz")
			err = err.ViaKey("E").ViaIndex(0).ViaField("jar")
			return err
		}(),
		want: "invalid field(s): jar[0][E].baz[1].bar[A].faa, jar[0][E].baz[1].bar[A].foo",
	}, {
		name:     "nil propagation",
		err:      nil,
		prefixes: [][]string{{"baz", "ugh", "INDEX:0", "KEY:A"}},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fe := test.err
			// Simulate propagation up a call stack.
			for _, prefix := range test.prefixes {
				for _, p := range prefix {
					if strings.HasPrefix(p, "INDEX") {
						index := strings.Split(p, ":")
						fe = fe.ViaIndex(makeIndex(index[1]))
					} else if strings.HasPrefix(p, "FIELDINDEX") {
						index := strings.Split(p, ":")
						fe = fe.ViaFieldIndex(makeFieldIndex(index[1]))
					} else if strings.HasPrefix(p, "KEY") {
						key := strings.Split(p, ":")
						fe = fe.ViaKey(makeKey(key[1]))
					} else if strings.HasPrefix(p, "FIELDKEY") {
						index := strings.Split(p, ":")
						fe = fe.ViaFieldKey(makeFieldKey(index[1]))
					} else {
						fe = fe.ViaField(p)
					}
				}
			}

			if test.want != "" {
				got := fe.Error()
				if got != test.want {
					t.Errorf("%s: Error() = %v, wanted %v", test.name, got, test.want)
				}
			} else if fe != nil {
				t.Errorf("%s: ViaField() = %v, wanted nil", test.name, fe)
			}
		})
	}
}

func TestNilError(t *testing.T) {
	var err *FieldError
	got := err.Error()
	want := ""
	if got != want {
		t.Errorf("got %v, wanted %v", got, want)
	}
}

func TestAlso(t *testing.T) {
	tests := []struct {
		name     string
		err      *FieldError
		also     []FieldError
		prefixes [][]string
		want     string
	}{{
		name: "nil",
		err:  nil,
		also: []FieldError{{
			Message: "also this",
			Paths:   []string{"woo"},
		}},
		prefixes: [][]string{{"foo"}},
		want:     "also this: foo.woo",
	}, {
		name: "nil all the way",
		err:  nil,
		also: []FieldError{{}},
		want: "",
	}, {
		name: "simple",
		err: &FieldError{
			Message: "hear me roar",
			Paths:   []string{"bar"},
		},
		also: []FieldError{{
			Message: "also this",
			Paths:   []string{"woo"},
		}},
		prefixes: [][]string{{"foo", "[A]", "[B]", "[C]"}},
		want: `also this: foo[A][B][C].woo
hear me roar: foo[A][B][C].bar`,
	}, {
		name: "lots of also",
		err: &FieldError{
			Message: "knock knock",
			Paths:   []string{"foo"},
		},
		also: []FieldError{{
			Message: "also this",
			Paths:   []string{"A"},
		}, {
			Message: "and this",
			Paths:   []string{"B"},
		}, {
			Message: "not without this",
			Paths:   []string{"C"},
		}},
		prefixes: [][]string{{"bar"}},
		want: `also this: bar.A
and this: bar.B
knock knock: bar.foo
not without this: bar.C`,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fe := test.err

			for _, err := range test.also {
				fe = fe.Also(&err)
			}

			// Simulate propagation up a call stack.
			for _, prefix := range test.prefixes {
				fe = fe.ViaField(prefix...)
			}

			if test.want != "" {
				got := fe.Error()
				if got != test.want {
					t.Errorf("%s: Error() = %v, wanted %v", test.name, got, test.want)
				}
			} else if fe != nil {
				t.Errorf("%s: ViaField() = %v, wanted nil", test.name, fe)
			}
		})
	}
}

func TestAlsoStaysNil(t *testing.T) {
	var err *FieldError
	if err != nil {
		t.Errorf("expected nil, got %v, wanted nil", err)
	}

	err = err.Also(nil)
	if err != nil {
		t.Errorf("expected nil, got %v, wanted nil", err)
	}

	err = err.ViaField("nil").Also(nil)
	if err != nil {
		t.Errorf("expected nil, got %v, wanted nil", err)
	}

	err = err.Also(&FieldError{})
	if err != nil {
		t.Errorf("expected nil, got %v, wanted nil", err)
	}
}

func TestFlattenFieldErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      *FieldError
		also     []FieldError
		prefixes [][]string
		want     string
	}{{
		name: "simple",
		err: &FieldError{
			Message: "A simple error message",
			Paths:   []string{"bar"},
		},
		also: []FieldError{{
			Message: "A simple error message",
			Paths:   []string{"foo"},
		}},
		want: `A simple error message: bar, foo`,
	}, {
		name: "conflict",
		err: &FieldError{
			Message: "A simple error message",
			Paths:   []string{"bar", "foo"},
		},
		also: []FieldError{{
			Message: "A simple error message",
			Paths:   []string{"foo"},
		}},
		want: `A simple error message: bar, foo`,
	}, {
		name: "lots of also",
		err: (&FieldError{
			Message: "this error",
			Paths:   []string{"bar", "foo"},
		}).Also(&FieldError{
			Message: "another",
			Paths:   []string{"right", "left"},
		}).ViaField("head"),
		also: []FieldError{{
			Message: "An alpha error message",
			Paths:   []string{"A"},
		}, {
			Message: "An alpha error message",
			Paths:   []string{"B"},
		}, {
			Message: "An alpha error message",
			Paths:   []string{"B"},
		}, {
			Message: "An alpha error message",
			Paths:   []string{"B"},
		}, {
			Message: "An alpha error message",
			Paths:   []string{"B"},
		}, {
			Message: "An alpha error message",
			Paths:   []string{"B"},
		}, {
			Message: "An alpha error message",
			Paths:   []string{"B"},
		}, {
			Message: "An alpha error message",
			Paths:   []string{"C"},
		}, {
			Message: "An alpha error message",
			Paths:   []string{"D"},
		}, {
			Message: "this error",
			Paths:   []string{"foo"},
			Details: "devil is in the details",
		}, {
			Message: "this error",
			Paths:   []string{"foo"},
			Details: "more details",
		}},
		prefixes: [][]string{{"this"}},
		want: `An alpha error message: this.A, this.B, this.C, this.D
another: this.head.left, this.head.right
this error: this.head.bar, this.head.foo
this error: this.foo
devil is in the details
this error: this.foo
more details`,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fe := test.err

			for _, err := range test.also {
				fe = fe.Also(&err)
			}

			// Simulate propagation up a call stack.
			for _, prefix := range test.prefixes {
				fe = fe.ViaField(prefix...)
			}

			if test.want != "" {
				got := fe.Error()
				if got != test.want {
					t.Errorf("%s: Error() = %v, wanted %v", test.name, got, test.want)
				}
			} else if fe != nil {
				t.Errorf("%s: ViaField() = %v, wanted nil", test.name, fe)
			}
		})
	}
}

func TestFlatten(t *testing.T) {
	tests := []struct {
		name    string
		indices []string
		want    string
	}{{
		name:    "simple",
		indices: strings.Split("foo.[1]", "."),
		want:    "foo[1]",
	}, {
		name:    "no brackets",
		indices: strings.Split("foo.bar", "."),
		want:    "foo.bar",
	}, {
		name:    "err([0]).ViaField(bar).ViaField(foo)",
		indices: strings.Split("foo.bar.[0]", "."),
		want:    "foo.bar[0]",
	}, {
		name:    "err(bar).ViaIndex(0).ViaField(foo)",
		indices: strings.Split("foo.[0].bar", "."),
		want:    "foo[0].bar",
	}, {
		name:    "err(bar).ViaField(foo).ViaIndex(0)",
		indices: strings.Split("[0].foo.bar", "."),
		want:    "[0].foo.bar",
	}, {
		name:    "err(bar).ViaIndex(0).ViaIndex[1].ViaField(foo)",
		indices: strings.Split("foo.[1].[0].bar", "."),
		want:    "foo[1][0].bar",
	}, {
		name:    "err(bar).ViaIndex(0).ViaIndex[1].ViaField(foo)",
		indices: []string{"foo", "bar.[0].baz"},
		want:    "foo.bar[0].baz",
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			got := flatten(test.indices)

			if got != test.want {
				t.Errorf("got: %q, want %q", got, test.want)
			}
		})
	}
}

func makeIndex(index string) int {
	all := strings.Split(index, ",")
	if i, err := strconv.Atoi(all[0]); err == nil {
		return i
	}
	return -1
}

func makeFieldIndex(fi string) (string, int) {
	all := strings.Split(fi, ",")
	if i, err := strconv.Atoi(all[1]); err == nil {
		return all[0], i
	}
	return "error", -1
}

func makeKey(key string) string {
	all := strings.Split(key, ",")
	return all[0]
}

func makeFieldKey(fk string) (string, string) {
	all := strings.Split(fk, ",")
	return all[0], all[1]
}

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

//
//
//func TestViaIndexOrKeyFieldError(t *testing.T) {
//	tests := []struct {
//		name     string
//		err      *FieldError
//		prefixes [][]string
//		want     string
//	}{{
//		name: "simple single no propagation",
//		err: Error{
//			Message: "hear me roar",
//			Paths:   []string{"bar"},
//		}.Wrap(),
//		prefixes: [][]string{{"INDEX:3", "INDEX:2", "INDEX:1", "foo"}},
//		want:     "hear me roar: foo[1][2][3].bar",
//	}, {
//		name: "simple key",
//		err: Error{
//			Message: "hear me roar",
//			Paths:   []string{"bar"},
//		}.Wrap(),
//		prefixes: [][]string{{"KEY:C", "KEY:B", "KEY:A", "foo"}},
//		want:     "hear me roar: foo[A][B][C].bar",
//	}, {
//		name:     "missing field propagation",
//		err:      ErrMissingField("foo", "bar"),
//		prefixes: [][]string{{"[2]", "baz"}},
//		want:     "missing field(s): baz[2].foo, baz[2].bar",
//	}, {
//		name: "invalid key name",
//		err: ErrInvalidKeyName("b@r", "name",
//			"can not use @", "do not try"),
//		prefixes: [][]string{{"baz", "INDEX:0", "foo"}},
//		want: `invalid key name "b@r": foo[0].baz.name
//can not use @, do not try`,
//	}, {
//		name: "invalid key name with keys",
//		err: ErrInvalidKeyName("b@r", "name",
//			"can not use @", "do not try"),
//		prefixes: [][]string{{"baz", "INDEX:0", "foo"}, {"bar", "KEY:A", "boo"}},
//		want: `invalid key name "b@r": boo[A].bar.foo[0].baz.name
//can not use @, do not try`,
//	}, {
//		name: "multi prefixes provided",
//		err: Error{
//			Message: "invalid field(s)",
//			Paths:   []string{"foo"},
//		}.Wrap(),
//		prefixes: [][]string{{"INDEX:2"}, {"bee"}, {"INDEX:0"}, {"baa", "baz", "ugh"}},
//		want:     "invalid field(s): ugh.baz.baa[0].bee[2].foo",
//	}, {
//		name: "use helper viaFieldIndex",
//		err: Error{
//			Message: "invalid field(s)",
//			Paths:   []string{"foo"},
//		}.Wrap(),
//		prefixes: [][]string{{"FIELDINDEX:bee,2"}, {"FIELDINDEX:baa,0"}, {"baz", "ugh"}},
//		want:     "invalid field(s): ugh.baz.baa[0].bee[2].foo",
//	}, {
//		name: "use helper viaFieldKey",
//		err: Error{
//			Message: "invalid field(s)",
//			Paths:   []string{"foo"},
//		}.Wrap(),
//		prefixes: [][]string{{"FIELDKEY:bee,AAA"}, {"FIELDKEY:baa,BBB"}, {"baz", "ugh"}},
//		want:     "invalid field(s): ugh.baz.baa[BBB].bee[AAA].foo",
//	}, {
//		name: "bypass helpers",
//		err: Error{
//			Message: "invalid field(s)",
//			Paths:   []string{"foo"},
//		}.Wrap(),
//		prefixes: [][]string{{"[2]"}, {"[1]"}, {"bar"}},
//		want:     "invalid field(s): bar[1][2].foo",
//	}, {
//		name: "multi paths provided",
//		err: Error{
//			Message: "invalid field(s)",
//			Paths:   []string{"foo", "bar"},
//		}.Wrap(),
//		prefixes: [][]string{{"INDEX:0"}, {"index"}, {"KEY:A"}, {"map"}},
//		want:     "invalid field(s): map[A].index[0].foo, map[A].index[0].bar",
//	}, {
//		name: "manual index",
//		err: func() *FieldError {
//			// Example, return an error in a loop:
//			// for i, item := spec.myList {
//			//   err := item.validate().ViaIndex(i).ViaField("myList")
//			//   if err != nil {
//			// 		return err
//			//   }
//			// }
//			// --> I expect path to be myList[i].foo
//
//			err := Error{
//				Message: "invalid field(s)",
//				Paths:   []string{"foo"},
//			}.Wrap()
//
//			err = err.ViaIndex(0).ViaField("bar")
//			err = err.ViaIndex(2).ViaIndex(1).ViaField("baz")
//			err = err.ViaIndex(3).ViaIndex(4).ViaField("boof")
//			return err
//		}(),
//		want: "invalid field(s): boof[4][3].baz[1][2].bar[0].foo",
//	}, {
//		name: "manual multiple index",
//		err: func() *FieldError {
//
//			err := &Error{
//				Message: "invalid field(s)",
//				Paths:   []string{"foo"},
//			}
//
//			err = err.ViaField("bear", "[1]", "[2]", "[3]", "baz", "]xxx[").ViaField("bar")
//			return err.Wrap()
//		}(),
//		want: "invalid field(s): bar.bear[1][2][3].baz.]xxx[.foo",
//	}, {
//		name: "manual keys",
//		err: func() *FieldError {
//			err := &Error{
//				Message: "invalid field(s)",
//				Paths:   []string{"foo"},
//			}
//
//			err = err.ViaKey("A").ViaField("bar")
//			err = err.ViaKey("CCC").ViaKey("BB").ViaField("baz")
//			err = err.ViaKey("E").ViaKey("F").ViaField("jar")
//			return err.Wrap()
//		}(),
//		want: "invalid field(s): jar[F][E].baz[BB][CCC].bar[A].foo",
//	}, {
//		name: "manual index and keys",
//		err: func() *FieldError {
//			err := &Error{
//				Message: "invalid field(s)",
//				Paths:   []string{"foo", "faa"},
//			}
//
//			err = err.ViaKey("A").ViaField("bar")
//			err = err.ViaIndex(1).ViaField("baz")
//			err = err.ViaKey("E").ViaIndex(0).ViaField("jar")
//			return err.Wrap()
//		}(),
//		want: "invalid field(s): jar[0][E].baz[1].bar[A].foo, jar[0][E].baz[1].bar[A].faa",
//	}, {
//		name:     "nil propagation",
//		err:      nil,
//		prefixes: [][]string{{"baz", "ugh", "INDEX:0", "KEY:A"}},
//	}}
//
//	for _, test := range tests {
//		t.Run(test.name, func(t *testing.T) {
//			fe := test.err
//			// Simulate propagation up a call stack.
//			for _, prefix := range test.prefixes {
//				for _, p := range prefix {
//					if strings.HasPrefix(p, "INDEX") {
//						index := strings.Split(p, ":")
//						fe = fe.ViaIndex(makeIndex(index[1]))
//					} else if strings.HasPrefix(p, "FIELDINDEX") {
//						index := strings.Split(p, ":")
//						fe = fe.ViaFieldIndex(makeFieldIndex(index[1]))
//					} else if strings.HasPrefix(p, "KEY") {
//						key := strings.Split(p, ":")
//						fe = fe.ViaKey(makeKey(key[1]))
//					} else if strings.HasPrefix(p, "FIELDKEY") {
//						index := strings.Split(p, ":")
//						fe = fe.ViaFieldKey(makeFieldKey(index[1]))
//					} else {
//						fe = fe.ViaField(p)
//					}
//				}
//			}
//
//			if test.want != "" {
//				got := fe.Error()
//				if got != test.want {
//					t.Errorf("Error() = %v, wanted %v", got, test.want)
//				}
//			} else if fe != nil {
//				t.Errorf("ViaField() = %v, wanted nil", fe)
//			}
//		})
//	}
//}

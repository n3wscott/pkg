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
	"fmt"
	"sort"
	"strings"
)

// CurrentField is a constant to supply as a fieldPath for when there is
// a problem with the current field itself.
const CurrentField = ""

// FieldError is used to propagate the context of errors pertaining to
// specific fields in a manner suitable for use in a recursive walk, so
// that errors contain the appropriate field context.
// FieldError methods are non-mutating.
// +k8s:deepcopy-gen=false
type FieldError struct {
	Message string
	Paths   []string
	// Details contains an optional longer payload.
	// +optional
	Details string
	errors  map[string]FieldError
}

// FieldError implements error
var _ error = (*FieldError)(nil)

// ViaField is used to propagate a validation error along a field access.
// For example, if a type recursively validates its "spec" via:
//   if err := foo.Spec.Validate(); err != nil {
//     // Augment any field paths with the context that they were accessed
//     // via "spec".
//     return err.ViaField("spec")
//   }
func (fe *FieldError) ViaField(prefix ...string) *FieldError {
	if fe == nil {
		return nil
	}
	newErr := &FieldError{}
	for _, e := range fe.getNormalizedErrors() {
		// Prepend the Prefix to existing errors.
		newPaths := make([]string, 0, len(e.Paths))
		for _, oldPath := range e.Paths {
			newPaths = append(newPaths, flatten(append(prefix, oldPath)))
		}
		e.Paths = newPaths

		// Append the mutated error to the errors list.
		newErr = newErr.Also(&e)
	}
	return newErr
}

// ViaIndex is used to attach an index to the next ViaField provided.
// For example, if a type recursively validates a parameter that has a collection:
//  for i, c := range spec.Collection {
//    if err := doValidation(c); err != nil {
//      return err.ViaIndex(i).ViaField("collection")
//    }
//  }
func (fe *FieldError) ViaIndex(index int) *FieldError {
	return fe.ViaField(asIndex(index))
}

// ViaFieldIndex is the short way to chain: err.ViaIndex(bar).ViaField(foo)
func (fe *FieldError) ViaFieldIndex(field string, index int) *FieldError {
	return fe.ViaIndex(index).ViaField(field)
}

// ViaKey is used to attach a key to the next ViaField provided.
// For example, if a type recursively validates a parameter that has a collection:
//  for k, v := range spec.Bag. {
//    if err := doValidation(v); err != nil {
//      return err.ViaKey(k).ViaField("bag")
//    }
//  }
func (fe *FieldError) ViaKey(key string) *FieldError {
	return fe.ViaField(asKey(key))
}

// ViaFieldKey is the short way to chain: err.ViaKey(bar).ViaField(foo)
func (fe *FieldError) ViaFieldKey(field string, key string) *FieldError {
	return fe.ViaKey(key).ViaField(field)
}

// also collects errors, returns a new collection of existing errors and new errors.
func (fe *FieldError) Also(errs ...*FieldError) *FieldError {
	newErrs := &FieldError{}
	// collect the current objects errors, if it has any
	if fe != nil {
		newErrs.errors = fe.getNormalizedErrors()
	}
	// and then collect the passed in errors
	for _, e := range errs {
		k := key(e)
		if err, ok := newErrs.errors[k]; ok {
			// merge the keys.
			newErr := newErrs.errors[k]
			newErr.Paths = mergePaths(newErr.Paths, err.Paths)
			newErrs.errors[k] = newErr
		} else {
			newErrs.errors[k] = err
		}
	}
	if len(newErrs.errors) == 0 {
		return nil
	}
	return newErrs
}

func (fe *FieldError) getNormalizedErrors() map[string]FieldError {
	// in case we call getNormalizedErrors on a nil object, return just an empty
	// list. This can happen when .Error() is called on a nil object.
	if fe == nil {
		return map[string]FieldError(nil)
	}
	errors := make(map[string]FieldError, len(fe.errors))
	// if this FieldError is a leaf,
	if fe.Message != "" {
		err := FieldError{
			Message: fe.Message,
			Paths:   fe.Paths,
			Details: fe.Details,
		}
		errors[key(&err)] = err

	}
	// and then collect all other errors recursively.
	for _, e := range fe.errors {
		for k, err := range e.getNormalizedErrors() {
			if v, ok := errors[k]; ok {
				// merge the keys.
				v.Paths = mergePaths(v.Paths, err.Paths) // TODO: this is hard.
				errors[k] = v
			} else {
				errors[k] = err
			}
		}
	}

	return errors
}

func mergePaths(a, b []string) []string {
	newPaths := make([]string, 0, len(a)+len(b))
	newPaths = append(newPaths, a...)
	for p := 0; p < len(b); p++ {
		if containsString(newPaths, b[p]) == false {
			newPaths = append(newPaths, b[p])
		}
	}
	return newPaths
}

func asIndex(index int) string {
	return fmt.Sprintf("[%d]", index)
}

func asKey(key string) string {
	return fmt.Sprintf("[%s]", key)
}

// merge will combine errors that differ only in their paths.
// This assumes that errors has been normalized.
func merge(errors []FieldError) []FieldError {
	if len(errors) <= 1 {
		return errors
	}

	//ignoreArguments := cmpopts.IgnoreFields(FieldError{}, "Paths")
	//ignoreUnexported := cmpopts.IgnoreUnexported(FieldError{})

	// Sort first.
	sort.Slice(errors, func(i, j int) bool { return errors[i].Message < errors[j].Message })

	newErrors := make([]FieldError, 0, len(errors))
	newErrors = append(newErrors, errors[0])
	curr := 0
	for i := 1; i < len(errors); i++ {
		if newErrors[curr].Message == errors[i].Message && newErrors[curr].Details == errors[i].Details {
			//if diff := cmp.Diff(newErrors[curr], errors[i], ignoreArguments, ignoreUnexported); diff == "" {
			// they match, merge the paths.
			nextPaths := make([]string, 0, len(errors[i].Paths)+len(newErrors[curr].Paths))
			for p := 0; p < len(errors[i].Paths); p++ {
				// Check that the path that is about to be appended is not
				// already in the list
				if containsString(newErrors[curr].Paths, errors[i].Paths[p]) == false {
					nextPaths = append(nextPaths, errors[i].Paths[p])
				}
			}
			// only add new ones.
			newErrors[curr].Paths = append(newErrors[curr].Paths, nextPaths...)
		} else {
			// moving the pointer, save the current object.
			newErrors = append(newErrors, errors[i])
			curr = len(newErrors) - 1
		}
	}
	return newErrors
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// flatten takes in a array of path components and looks for chances to flatten
// objects that have index prefixes, examples:
//   err([0]).ViaField(bar).ViaField(foo) -> foo.bar.[0] converts to foo.bar[0]
//   err(bar).ViaIndex(0).ViaField(foo) -> foo.[0].bar converts to foo[0].bar
//   err(bar).ViaField(foo).ViaIndex(0) -> [0].foo.bar converts to [0].foo.bar
//   err(bar).ViaIndex(0).ViaIndex[1].ViaField(foo) -> foo.[1].[0].bar converts to foo[1][0].bar
func flatten(path []string) string {
	var newPath []string
	for _, part := range path {
		for _, p := range strings.Split(part, ".") {
			if p == CurrentField {
				continue
			} else if len(newPath) > 0 && isIndex(p) {
				newPath[len(newPath)-1] = fmt.Sprintf("%s%s", newPath[len(newPath)-1], p)
			} else {
				newPath = append(newPath, p)
			}
		}
	}
	return strings.Join(newPath, ".")
}

func isIndex(part string) bool {
	return strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]")
}

// key returns the key that should be used for a given FieldError for the
// internal map that stores errors.
func key(err *FieldError) string {
	return fmt.Sprintf("%s-%s", err.Message, err.Details)
}

// Error implements error
func (fe *FieldError) Error() string {
	var errs []string

	errors := make([]FieldError, 0, len(fe.errors))
	for _, e := range fe.getNormalizedErrors() {
		errors = append(errors, e)
	}
	sort.Slice(errors, func(i, j int) bool { return errors[i].Message < errors[j].Message })

	for _, e := range errors {
		sort.Slice(e.Paths, func(i, j int) bool { return e.Paths[i] < e.Paths[j] })
		if e.Details == "" {
			errs = append(errs, fmt.Sprintf("%v: %v", e.Message, strings.Join(e.Paths, ", ")))
		} else {
			errs = append(errs, fmt.Sprintf("%v: %v\n%v", e.Message, strings.Join(e.Paths, ", "), e.Details))
		}
	}
	return strings.Join(errs, "\n")
}

// ErrMissingField is a variadic helper method for constructing a FieldError for
// a set of missing fields.
func ErrMissingField(fieldPaths ...string) *FieldError {
	return &FieldError{
		Message: "missing field(s)",
		Paths:   fieldPaths,
	}
}

// ErrDisallowedFields is a variadic helper method for constructing a FieldError
// for a set of disallowed fields.
func ErrDisallowedFields(fieldPaths ...string) *FieldError {
	return &FieldError{
		Message: "must not set the field(s)",
		Paths:   fieldPaths,
	}
}

// ErrInvalidValue constructs a FieldError for a field that has received an
// invalid string value.
func ErrInvalidValue(value, fieldPath string) *FieldError {
	return &FieldError{
		Message: fmt.Sprintf("invalid value %q", value),
		Paths:   []string{fieldPath},
	}
}

// ErrMissingOneOf is a variadic helper method for constructing a FieldError for
// not having at least one field in a mutually exclusive field group.
func ErrMissingOneOf(fieldPaths ...string) *FieldError {
	return &FieldError{
		Message: "expected exactly one, got neither",
		Paths:   fieldPaths,
	}
}

// ErrMultipleOneOf is a variadic helper method for constructing a FieldError
// for having more than one field set in a mutually exclusive field group.
func ErrMultipleOneOf(fieldPaths ...string) *FieldError {
	return &FieldError{
		Message: "expected exactly one, got both",
		Paths:   fieldPaths,
	}
}

// ErrInvalidKeyName is a variadic helper method for constructing a FieldError
// that specifies a key name that is invalid.
func ErrInvalidKeyName(value, fieldPath string, details ...string) *FieldError {
	return &FieldError{
		Message: fmt.Sprintf("invalid key name %q", value),
		Paths:   []string{fieldPath},
		Details: strings.Join(details, ", "),
	}
}

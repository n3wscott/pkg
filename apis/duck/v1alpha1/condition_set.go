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

package v1alpha1

import (
	"reflect"
	"sort"
	"time"

	"fmt"

	"github.com/knative/pkg/apis"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Conditions is the interface for a Resource that implements the getter and
// setter for accessing a Condition collection.
// +k8s:deepcopy-gen=true
type ConditionsAccessor interface {
	GetConditions() Conditions
	SetConditions(Conditions)
}

// ConditionSet is an abstract collection of the possible ConditionType values
// that a particular resource might expose.  It also holds the "happy condition"
// for that resource, which we define to be one of Ready or Succeeded depending
// on whether it is a Living or Batch process respectively.
// +k8s:deepcopy-gen=false
type ConditionSet struct {
	happy      ConditionType
	dependents []ConditionType
}

// ConditionManager allows a resource to operate on its Conditions using higher
// order operations.
type ConditionManager interface {
	// IsHappy looks at the happy condition and returns true if that condition is
	// set to true.
	IsHappy() bool

	// GetCondition finds and returns the Condition that matches the ConditionType
	// previously set on Conditions.
	GetCondition(t ConditionType) *Condition

	// SetCondition sets or updates the Condition on Conditions for Condition.Type.
	// If there is an update, Conditions are stored back sorted.
	SetCondition(new Condition)

	// MarkTrue sets the status of t to true, and then marks the happy condition to
	// true if all other dependents are also true.
	MarkTrue(t ConditionType)

	// MarkUnknown sets the status of t to Unknown and also sets the happy condition
	// to Unknown if no other dependent condition is in an error state.
	MarkUnknown(t ConditionType, reason, messageFormat string, messageA ...interface{})

	// MarkFalse sets the status of t and the happy condition to False.
	MarkFalse(t ConditionType, reason, messageFormat string, messageA ...interface{})

	// InitializeConditions updates all Conditions in the ConditionSet to Unknown
	// if not set.
	InitializeConditions()

	// InitializeCondition updates a Condition to Unknown if not set.
	InitializeCondition(t ConditionType)
}

// NewLivingConditionSet returns a ConditionSet to hold the conditions for the
// ongoing resource. ConditionReady is used as the happy.
func NewLivingConditionSet(d ...ConditionType) ConditionSet {
	return newConditionSet(ConditionReady, d...)
}

// NewBatchConditionSet returns a ConditionSet to hold the conditions for the
// run once resource. ConditionSucceeded is used as the happy.
func NewBatchConditionSet(d ...ConditionType) ConditionSet {
	return newConditionSet(ConditionSucceeded, d...)
}

// newConditionSet returns a ConditionSet to hold the conditions that are
// important for the caller. The first ConditionType is the overarching status
// for that will be used to signal the resources' status is Ready or Succeeded.
func newConditionSet(happy ConditionType, dependents ...ConditionType) ConditionSet {
	var deps []ConditionType
	for _, d := range dependents {
		// Skip duplicates
		if d == happy || contains(deps, d) {
			continue
		}
		deps = append(deps, d)
	}
	return ConditionSet{
		happy:      happy,
		dependents: deps,
	}
}

func contains(ct []ConditionType, t ConditionType) bool {
	for _, c := range ct {
		if c == t {
			return true
		}
	}
	return false
}

// Check that conditionsImpl implements ConditionManager.
var _ ConditionManager = (*conditionsImpl)(nil)

// conditionsImpl implements the helper methods for evaluating Conditions.
// +k8s:deepcopy-gen=false
type conditionsImpl struct {
	ConditionSet
	accessor ConditionsAccessor
}

// Manage creates a ConditionManager from an object that implements
// ConditionsAccessopr using the original ConditionSet as a reference.
func (r ConditionSet) Manage(accessor ConditionsAccessor) ConditionManager {
	return conditionsImpl{
		accessor:     accessor,
		ConditionSet: r,
	}
}

// IsHappy looks at the happy condition and returns true if that condition is
// set to true.
func (r conditionsImpl) IsHappy() bool {
	if c := r.GetCondition(r.happy); c == nil || !c.IsTrue() {
		return false
	}
	return true
}

// GetCondition finds and returns the Condition that matches the ConditionType
// previously set on Conditions.
func (r conditionsImpl) GetCondition(t ConditionType) *Condition {
	if r.accessor == nil {
		return nil
	}
	for _, c := range r.accessor.GetConditions() {
		if c.Type == t {
			return &c
		}
	}
	return nil
}

// SetCondition sets or updates the Condition on Conditions for Condition.Type.
// If there is an update, Conditions are stored back sorted.
func (r conditionsImpl) SetCondition(new Condition) {
	if r.accessor == nil {
		return
	}
	t := new.Type
	var conditions Conditions
	for _, c := range r.accessor.GetConditions() {
		if c.Type != t {
			conditions = append(conditions, c)
		} else {
			// If we'd only update the LastTransitionTime, then return.
			new.LastTransitionTime = c.LastTransitionTime
			if reflect.DeepEqual(&new, &c) {
				return
			}
		}
	}
	new.LastTransitionTime = apis.VolatileTime{Inner: metav1.NewTime(time.Now())}
	conditions = append(conditions, new)
	// Sorted for convince of the consumer, i.e.: kubectl.
	sort.Slice(conditions, func(i, j int) bool { return conditions[i].Type < conditions[j].Type })
	r.accessor.SetConditions(conditions)
}

// MarkTrue sets the status of t to true, and then marks the happy condition to
// true if all other dependents are also true.
func (r conditionsImpl) MarkTrue(t ConditionType) {
	// set the specified condition
	r.SetCondition(Condition{
		Type:   t,
		Status: corev1.ConditionTrue,
	})

	// check the dependents.
	for _, cond := range r.dependents {
		c := r.GetCondition(cond)
		// Failed or Unknown conditions trump true conditions
		if !c.IsTrue() {
			return
		}
	}

	// set the happy condition
	r.SetCondition(Condition{
		Type:   r.happy,
		Status: corev1.ConditionTrue,
	})
}

// MarkUnknown sets the status of t to Unknown and also sets the happy condition
// to Unknown if no other dependent condition is in an error state.
func (r conditionsImpl) MarkUnknown(t ConditionType, reason, messageFormat string, messageA ...interface{}) {
	// set the specified condition
	r.SetCondition(Condition{
		Type:    t,
		Status:  corev1.ConditionUnknown,
		Reason:  reason,
		Message: fmt.Sprintf(messageFormat, messageA...),
	})

	// check the dependents.
	for _, cond := range r.dependents {
		c := r.GetCondition(cond)
		// Failed conditions trump Unknown conditions
		if c.IsFalse() {
			// Double check that the happy condition is also false.
			happy := r.GetCondition(r.happy)
			if !happy.IsFalse() {
				r.MarkFalse(r.happy, reason, messageFormat, messageA)
			}
			return
		}
	}

	// set the happy condition
	r.SetCondition(Condition{
		Type:    r.happy,
		Status:  corev1.ConditionUnknown,
		Reason:  reason,
		Message: fmt.Sprintf(messageFormat, messageA...),
	})
}

// MarkFalse sets the status of t and the happy condition to False.
func (r conditionsImpl) MarkFalse(t ConditionType, reason, messageFormat string, messageA ...interface{}) {
	for _, t := range []ConditionType{
		t,
		r.happy,
	} {
		r.SetCondition(Condition{
			Type:    t,
			Status:  corev1.ConditionFalse,
			Reason:  reason,
			Message: fmt.Sprintf(messageFormat, messageA...),
		})
	}
}

// InitializeConditions updates all Conditions in the ConditionSet to Unknown
// if not set.
func (r conditionsImpl) InitializeConditions() {
	for _, t := range append(r.dependents, r.happy) {
		r.InitializeCondition(t)
	}
}

// InitializeCondition updates a Condition to Unknown if not set.
func (r conditionsImpl) InitializeCondition(t ConditionType) {
	if c := r.GetCondition(t); c == nil {
		r.SetCondition(Condition{
			Type:   t,
			Status: corev1.ConditionUnknown,
		})
	}
}

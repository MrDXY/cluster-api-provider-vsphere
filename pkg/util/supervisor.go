/*
Copyright 2021 The Kubernetes Authors.

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

package util

import (
	"fmt"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	infrav1 "sigs.k8s.io/cluster-api-provider-vsphere/apis/v1beta1"
	vmwarev1 "sigs.k8s.io/cluster-api-provider-vsphere/apis/vmware/v1beta1"
)

// IsSupervisorType identifies whether the passed object is using the supervisor API.
func IsSupervisorType(input interface{}) (bool, error) {
	switch input.(type) {
	case *infrav1.VSphereCluster, *infrav1.VSphereMachine:
		return false, nil
	case *vmwarev1.VSphereCluster, *vmwarev1.VSphereMachine:
		return true, nil
	default:
		return false, fmt.Errorf("unexpected type %s", reflect.TypeOf(input))
	}
}

// SetControllerReferenceWithOverride sets owner as a Controller OwnerReference on controlled.
// This is used for garbage collection of the controlled object and for
// reconciling the owner object on changes to controlled (with a Watch + EnqueueRequestForOwner).
// Since only one OwnerReference can be a controller, it returns an error if
// there is another OwnerReference with Controller flag set unless it was a legacy controller owner.
func SetControllerReferenceWithOverride(owner, controlled metav1.Object, scheme *runtime.Scheme) error {
	// Validate the owner.
	ro, ok := owner.(runtime.Object)
	if !ok {
		return fmt.Errorf("%T is not a runtime.Object, cannot call SetControllerReference", owner)
	}

	// Create a new controller ref.
	gvk, err := apiutil.GVKForObject(ro, scheme)
	if err != nil {
		return err
	}
	ref := metav1.OwnerReference{
		APIVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
		Name:       owner.GetName(),
	}

	deleteAllControllerRefs(controlled, ref)

	return controllerutil.SetControllerReference(owner, controlled, scheme)
}

// deleteAllControllerRefs Removes existing controller reference from controlled object.
func deleteAllControllerRefs(controlled metav1.Object, ref metav1.OwnerReference) {
	owners := controlled.GetOwnerReferences()
	for i := range owners {
		// We don't want controller references to be removed if they are the same object, to avoid
		// unnecessary patches.
		if owners[i].Controller != nil && *owners[i].Controller && !referSameObject(owners[i], ref) {
			owners = append(owners[:i], owners[i+1:]...)
			break
		}
	}
	controlled.SetOwnerReferences(owners)
}

// Returns true if a and b point to the same object.
func referSameObject(a, b metav1.OwnerReference) bool {
	aGV, err := schema.ParseGroupVersion(a.APIVersion)
	if err != nil {
		return false
	}

	bGV, err := schema.ParseGroupVersion(b.APIVersion)
	if err != nil {
		return false
	}

	return aGV.Group == bGV.Group && a.Kind == b.Kind && a.Name == b.Name
}

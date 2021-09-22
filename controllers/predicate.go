/*
Copyright 2021 The OpenYurt Authors.

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

package controllers

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/go-logr/logr"
	devicev1alpha1 "github.com/openyurtio/device-controller/api/v1alpha1"
)

func genFirstUpdateFilter(
	objKind string, log logr.Logger) predicate.Predicate {
	return predicate.Funcs{
		// ignore the update event that is generated due to a
		// new deviceprofile being added to the Edgex Foundry
		UpdateFunc: func(e event.UpdateEvent) bool {
			log := log.WithValues("kind", objKind)
			oldDp, ok := e.ObjectOld.(devicev1alpha1.EdgeXObject)
			if !ok {
				log.Info(
					"fail to assert object to deviceprofile")
				return false
			}
			newDp, ok := e.ObjectNew.(devicev1alpha1.EdgeXObject)
			if !ok {
				log.Info(
					"fail to assert object to deviceprofile")
				return false
			}
			if oldDp.IsAddedToEdgeX() == false &&
				newDp.IsAddedToEdgeX() == true {
				return false
			}
			return true
		},
	}
}

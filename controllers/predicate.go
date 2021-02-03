package controllers

import (
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	devicev1alpha1 "github.com/charleszheng44/device-controller/api/v1alpha1"
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

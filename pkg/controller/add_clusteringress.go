package controller

import (
	"github.com/bbrowning/knative-openshift-ingress/pkg/controller/clusteringress"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, clusteringress.Add)
}

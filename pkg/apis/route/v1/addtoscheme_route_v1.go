package v1

import (
	"github.com/bbrowning/knative-openshift-ingress/pkg/apis"
	routev1 "github.com/openshift/api/route/v1"
)

func init() {
	apis.AddToSchemes = append(apis.AddToSchemes, routev1.SchemeBuilder.AddToScheme)
}

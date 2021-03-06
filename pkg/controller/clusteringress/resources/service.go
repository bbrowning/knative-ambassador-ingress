package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/knative/pkg/kmeta"
	"github.com/knative/pkg/logging"
	"github.com/knative/serving/pkg/apis/networking"
	networkingv1alpha1 "github.com/knative/serving/pkg/apis/networking/v1alpha1"
	"github.com/knative/serving/pkg/apis/serving"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Mapping is an Ambassador Mapping
type Mapping struct {
	APIVersion        string            `json:"apiVersion"`
	Kind              string            `json:"kind"`
	Name              string            `json:"name"`
	Prefix            string            `json:"prefix"`
	PrefixRegex       bool              `json:"prefix_regex"`
	Service           string            `json:"service"`
	Weight            int               `json:"weight"`
	AddRequestHeaders map[string]string `json:"add_request_headers,omitempty"`
	Host              string            `json:"host"`
	HostRegex         bool              `json:"host_regex"`
	TimeoutMs         int64             `json:"timeout_ms"`
}

// MakeService makes a dummy Kubernetes Service used to provide the
// Ambassador config
func MakeService(ctx context.Context, ci *networkingv1alpha1.ClusterIngress, ambassadorNamespace string) (*corev1.Service, error) {
	logger := logging.FromContext(ctx)

	ambassadorYaml := ""
	for _, rule := range ci.Spec.Rules {
		hosts := rule.Hosts
		hostRegex := fmt.Sprintf("^(%s)$", strings.Join(hosts, "|"))
		for _, path := range rule.HTTP.Paths {
			prefix := path.Path
			prefixRegex := true
			if prefix == "" {
				prefix = "/"
				prefixRegex = false
			}
			for _, split := range path.Splits {
				service := fmt.Sprintf("%s.%s:%s", split.ServiceName, split.ServiceNamespace, split.ServicePort.String())
				mappingName := fmt.Sprintf("%s-%s", hosts[0], service)
				mapping := Mapping{
					APIVersion:        "ambassador/v1",
					Kind:              "Mapping",
					Name:              mappingName,
					Prefix:            prefix,
					PrefixRegex:       prefixRegex,
					Service:           service,
					Weight:            split.Percent,
					AddRequestHeaders: path.AppendHeaders,
					Host:              hostRegex,
					HostRegex:         true,
					TimeoutMs:         path.Timeout.Duration.Nanoseconds() / 1000000,
				}
				mappingYaml, err := yaml.Marshal(mapping)
				if err != nil {
					logger.Errorw("Error creating ambassador yaml", err)
					return nil, err
				}
				ambassadorYaml = fmt.Sprintf("%s---\n%s\n", ambassadorYaml, mappingYaml)
			}
		}
	}
	logger.Infof("Creating Ambassador Config:\n %s\n", ambassadorYaml)

	annotations := ci.ObjectMeta.Annotations
	annotations["getambassador.io/config"] = string(ambassadorYaml)

	labels := make(map[string]string)
	labels[networking.IngressLabelKey] = ci.Name

	ingressLabels := ci.Labels
	labels[serving.RouteLabelKey] = ingressLabels[serving.RouteLabelKey]
	labels[serving.RouteNamespaceLabelKey] = ingressLabels[serving.RouteNamespaceLabelKey]

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            ci.Name,
			Namespace:       ambassadorNamespace,
			OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(ci)},
			Labels:          labels,
			Annotations:     annotations,
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "None",
		},
	}, nil
}

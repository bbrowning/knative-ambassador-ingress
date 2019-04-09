package resources

import (
	"fmt"
	"strings"

	"github.com/cloudflare/cfssl/log"
	"github.com/ghodss/yaml"
	networkingv1alpha1 "github.com/knative/serving/pkg/apis/networking/v1alpha1"
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
func MakeService(ci *networkingv1alpha1.ClusterIngress) *corev1.Service {
	labels := map[string]string{
		"clusteringress": ci.Name,
	}

	ambassadorYaml := ""
	for _, rule := range ci.Spec.Rules {
		hosts := rule.Hosts
		hostRegex := fmt.Sprintf("^(%s)$", strings.Join(hosts, "|"))
		fmt.Printf("!!! HostRegex: %s\n", hostRegex)
		for _, path := range rule.HTTP.Paths {
			prefix := path.Path
			prefixRegex := true
			if prefix == "" {
				prefix = "/"
				prefixRegex = false
			}
			for _, split := range path.Splits {
				service := fmt.Sprintf("%s.%s:%s", split.ServiceName, split.ServiceNamespace, split.ServicePort.String())
				mapping := Mapping{
					APIVersion:        "ambassador/v1",
					Kind:              "Mapping",
					Name:              service,
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
					log.Error(err, "Error creating ambassador yaml")
				}
				ambassadorYaml = fmt.Sprintf("%s---\n%s\n", ambassadorYaml, mappingYaml)
			}
		}
	}
	fmt.Printf("!!! AMBASSADOR YAML:\n %s\n", ambassadorYaml)
	annotations := map[string]string{
		"getambassador.io/config": string(ambassadorYaml),
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        ci.Name + "-ambassador",
			Namespace:   "default",
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "None",
		},
	}
}

package utils

import (
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
)

// The Rook config requires the resources requirements in yaml format.
// Marshalling corev1.ResourceRequirements into yaml does not render the Quantity values as need.
// This type and the below helper function are used so that the resource configurations in
// reconcileRookCephOperatorConfig are marshalled properly.
type RookResourceRequirements struct {
	Name     string
	Resource corev1.ResourceRequirements `json:"resource,omitempty"`
}

type RookResourceRequirementsList []RookResourceRequirements

func MarshalRookResourceRequirements(reqList RookResourceRequirementsList) string {
	outList := make([]struct {
		Name     string `json:"name,omitempty"`
		Resource struct {
			Limits   map[corev1.ResourceName]string `json:"limits,omitempty"`
			Requests map[corev1.ResourceName]string `json:"requests,omitempty"`
		}
	}, len(reqList))

	for i := range reqList {
		in := &reqList[i]
		out := &outList[i]

		out.Name = in.Name
		for name, val := range in.Resource.Limits {
			if out.Resource.Limits == nil {
				out.Resource.Limits = map[corev1.ResourceName]string{}
			}
			out.Resource.Limits[name] = val.String()
		}
		for name, val := range in.Resource.Requests {
			if out.Resource.Requests == nil {
				out.Resource.Requests = map[corev1.ResourceName]string{}
			}
			out.Resource.Requests[name] = val.String()
		}
	}

	bytes, err := yaml.Marshal(outList)
	if err != nil {
		panic("Marshalling failed")
	}
	return string(bytes)
}

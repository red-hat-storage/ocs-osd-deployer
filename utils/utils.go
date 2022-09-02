/*


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

package utils

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Contains checks whether a string is contained within a slice
func Contains(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// Remove eliminates a given string from a slice and returns the new slice
func Remove(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

// AddLabel add a label to a resource metadata
func AddLabel(obj metav1.Object, key string, value string) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
		obj.SetLabels(labels)
	}
	labels[key] = value
}

func AddAnnotation(obj metav1.Object, key string, value string) {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
		obj.SetAnnotations(annotations)
	}
	annotations[key] = value
}

func MapItems(source []string, transform func(string) string) []string {
	target := make([]string, len(source))
	for i := 0; i < len(source); i += 1 {
		target[i] = transform(source[i])
	}
	return target
}

func Retry(attempts int, sleep time.Duration, f func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		if err = f(); err == nil {
			return nil
		}
		time.Sleep(sleep)
		log.Println("retrying after error:", err)
	}
	return fmt.Errorf("Failed after %d retries. Last error: %v", attempts, err)

}

func HTTPGetAndParseBody(endpoint string) (string, error) {
	resp, err := http.Get(endpoint)
	if err != nil {
		return "", fmt.Errorf("Failed to get %s: %v", endpoint, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Failed to read response body: %v", err)
	}
	return string(body), nil
}

func DeploymentNameFromPodName(podName string) (string, error) {
	//Pod names created from deployments follow the convention:
	// <deployment_name>-<pod-template-hash>-<uid>
	// Therefore, we can get the deployment_name by omitting the last hyphened two sections
	var i int
	if i = strings.LastIndex(podName, "-"); i == -1 {
		return "", fmt.Errorf("invalid pod name")
	}
	if i = strings.LastIndex(podName[0:i], "-"); i == -1 {
		return "", fmt.Errorf("invalid pod name")
	}
	return podName[0:i], nil
}

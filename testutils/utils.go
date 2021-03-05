package testutils

import (
	"context"
	"net/http"
	"time"

	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func WaitForResource(k8sClient client.Client, ctx context.Context, obj runtime.Object, timeout time.Duration, interval time.Duration) {
	key, err := client.ObjectKeyFromObject(obj)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	EventuallyWithOffset(1, func() bool {
		err := k8sClient.Get(ctx, key, obj)
		return err == nil
	}, timeout, interval).Should(BeTrue())
}

func EnsureNoResource(k8sClient client.Client, ctx context.Context, obj runtime.Object, timeout time.Duration, interval time.Duration) {
	key, err := client.ObjectKeyFromObject(obj)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	EventuallyWithOffset(1, func() bool {
		err := k8sClient.Get(ctx, key, obj)
		return err == nil
	}, timeout, interval).Should(BeFalse())
}

func GetResourceKey(obj runtime.Object) client.ObjectKey {
	key, err := client.ObjectKeyFromObject(obj)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return key
}

func ProbeReadiness() (int, error) {
	resp, err := http.Get("http://localhost:8081/readyz")
	if err != nil {
		return 0, err
	}
	return resp.StatusCode, nil
}

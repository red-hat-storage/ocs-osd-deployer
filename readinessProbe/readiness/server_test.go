package readiness

import (
	"context"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "github.com/openshift/ocs-osd-deployer/api/v1alpha1"
	utils "github.com/openshift/ocs-osd-deployer/testutils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("ManagedOCS Readiness Probe", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	When("managedocs reports its components as \"not ready\"", func() {
		BeforeEach(func() {
			ctx := context.Background()

			managedOCS := &v1.ManagedOCS{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ManagedOCSName,
					Namespace: TestNamespace,
				},
				Status: v1.ManagedOCSStatus{
					Components: v1.ComponentStatusMap{
						StorageCluster: v1.ComponentStatus{
							State: v1.ComponentPending,
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, managedOCS)).Should(Succeed())

			utils.WaitForResource(k8sClient, ctx, managedOCS, timeout, interval)
		})

		It("should cause the readiness logic to report \"not ready\"", func() {
			status, err := utils.ProbeReadiness()
			Expect(err).ToNot(HaveOccurred())
			Expect(status).To(Equal(http.StatusServiceUnavailable))
		})
	})

	When("managedocs reports its components as \"ready\"", func() {
		BeforeEach(func() {
			ctx := context.Background()
			managedOCS := &v1.ManagedOCS{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ManagedOCSName,
					Namespace: TestNamespace,
				},
			}

			// This test expects the ManagedOCS resource to already have been made.
			err := k8sClient.Get(ctx, utils.GetResourceKey(managedOCS), managedOCS)
			Expect(err).ToNot(HaveOccurred())

			managedOCS.Status.Components.StorageCluster.State = v1.ComponentReady

			Expect(k8sClient.Status().Update(ctx, managedOCS)).Should(Succeed())
		})

		It("should cause the readiness logic to report \"ready\"", func() {
			status, err := utils.ProbeReadiness()
			Expect(err).ToNot(HaveOccurred())
			Expect(status).To(Equal(http.StatusOK))
		})
	})
})

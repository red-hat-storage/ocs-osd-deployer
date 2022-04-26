package aws

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	utils "github.com/red-hat-storage/ocs-osd-deployer/testutils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var _ = Describe("AWS Data Gathering behavior", func() {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	log := ctrl.Log.WithName("aws-data-gather-test")
	ctx := context.Background()

	configMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DataConfigMapName,
			Namespace: testNamespace,
		},
	}

	When("aws data gathering successfully completes", func() {
		BeforeEach(func() {
			err := GatherData("http://"+testMockerAddr, k8sClient, testNamespace, log)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should have created a configmap with the VPC cidr", func() {
			err := k8sClient.Get(ctx, utils.GetResourceKey(&configMap), &configMap)
			Expect(err).ToNot(HaveOccurred())
			vpcCIDR, ok := configMap.Data[CidrKey]
			Expect(ok).To(Equal(true))
			Expect(vpcCIDR).To(Equal(fakeCIDR))
		})
	})

	When("The configmap aws-data already exists", func() {
		BeforeEach(func() {
			err := k8sClient.Get(ctx, utils.GetResourceKey(&configMap), &configMap)
			Expect(err).ToNot(HaveOccurred())
		})

		When("aws data gathering runs", func() {
			BeforeEach(func() {
				err := GatherData("http://"+testMockerAddr, k8sClient, testNamespace, log)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should have updated the configmap with the VPC cidr", func() {
				err := k8sClient.Get(ctx, utils.GetResourceKey(&configMap), &configMap)
				Expect(err).ToNot(HaveOccurred())
				vpcCIDR, ok := configMap.Data[CidrKey]
				Expect(ok).To(Equal(true))
				Expect(vpcCIDR).To(Equal(fakeCIDR))
			})
		})
	})
})

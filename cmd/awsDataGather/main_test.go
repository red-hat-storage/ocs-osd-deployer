package main

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/red-hat-storage/ocs-osd-deployer/pkg/aws"
	utils "github.com/red-hat-storage/ocs-osd-deployer/testutils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var _ = Describe("AWS Data Gathering behavior", func() {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	ctx := context.Background()

	configMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      aws.DataConfigMapName,
			Namespace: testNamespace,
		},
	}

	fakeDeployment := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-deployment",
			Namespace: testNamespace,
			UID:       "foobar",
		},
	}

	When("aws data gathering successfully completes", func() {
		var awsConfigMap corev1.ConfigMap

		BeforeEach(func() {
			err := gatherAndSaveData("http://"+testMockerAddr, fakeDeployment, k8sClient, context.TODO())
			Expect(err).ToNot(HaveOccurred())
		})

		It("should have created a configmap that is valid...", func() {
			err := k8sClient.Get(ctx, utils.GetResourceKey(&configMap), &awsConfigMap)
			Expect(err).ToNot(HaveOccurred())

			By("having the VPC cidr key")
			vpcCIDR, ok := awsConfigMap.Data[aws.CIDRKey]
			Expect(ok).To(Equal(true))
			Expect(vpcCIDR).To(Equal(fakeCIDR))

			By("having a deployment owner reference")
			Expect(len(awsConfigMap.OwnerReferences)).To(Equal(1))
		})

		AfterEach(func() {
			err := k8sClient.Delete(ctx, &awsConfigMap)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	When("The configmap aws-data already exists", func() {
		BeforeEach(func() {
			err := k8sClient.Create(ctx, &configMap)
			Expect(err).ToNot(HaveOccurred())
		})

		When("aws data gathering runs", func() {
			BeforeEach(func() {
				err := gatherAndSaveData("http://"+testMockerAddr, fakeDeployment, k8sClient, context.TODO())
				Expect(err).ToNot(HaveOccurred())
			})

			It("should have updated the configmap", func() {
				err := k8sClient.Get(ctx, utils.GetResourceKey(&configMap), &configMap)
				Expect(err).ToNot(HaveOccurred())

				By("adding VPC cidr key")
				vpcCIDR, ok := configMap.Data[aws.CIDRKey]
				Expect(ok).To(Equal(true))
				Expect(vpcCIDR).To(Equal(fakeCIDR))

				By("adding a deployment owner reference")
				Expect(len(configMap.OwnerReferences)).To(Equal(1))
			})
		})

		AfterEach(func() {
			err := k8sClient.Delete(ctx, &configMap)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

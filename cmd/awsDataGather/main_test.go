package main

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	tutils "github.com/red-hat-storage/ocs-osd-deployer/testutils"
	"github.com/red-hat-storage/ocs-osd-deployer/utils"
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
			Name:      utils.IMDSConfigMapName,
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
			err := k8sClient.Get(ctx, tutils.GetResourceKey(&configMap), &awsConfigMap)
			Expect(err).ToNot(HaveOccurred())

			By("having the VPC cidr key")
			vpcCIDR := awsConfigMap.Data
			fmt.Println(vpcCIDR)
			Expect(vpcCIDR).To(Equal(map[string]string{
				"0": "10.0.0.0/16",
				"1": "10.206.54.0/24",
			}))

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
				err := k8sClient.Get(ctx, tutils.GetResourceKey(&configMap), &configMap)
				Expect(err).ToNot(HaveOccurred())

				By("adding VPC cidr key")
				vpcCIDR := configMap.Data
				Expect(vpcCIDR).To(Equal(map[string]string{
					"0": "10.0.0.0/16",
					"1": "10.206.54.0/24",
				}))

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

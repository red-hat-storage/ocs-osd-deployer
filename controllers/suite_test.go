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

package controllers

import (
	"context"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	openshiftv1 "github.com/openshift/api/network/v1"
	opv1a1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	promv1a1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	ocsv1 "github.com/red-hat-storage/ocs-operator/api/v1"
	v1 "github.com/red-hat-storage/ocs-osd-deployer/api/v1alpha1"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

const (
	testPrimaryNamespace                       = "primary"
	testSecondaryNamespace                     = "secondary"
	testAddonParamsSecretName                  = "test-addon-secret"
	testPagerdutySecretName                    = "test-pagerduty-secret"
	testDeadMansSnitchSecretName               = "test-deadmanssnitch-secret"
	testSMTPSecretName                         = "test-smtp-secret"
	testAddonConfigMapName                     = "test-addon-configmap"
	testAddonConfigMapDeleteLabelKey           = "test-addon-configmap-delete-label-key"
	testDeployerCSVName                        = "ocs-osd-deployer.x.y.z"
	testGrafanaFederateSecretName              = "grafana-datasources"
	testK8sMetricsServiceMonitorAuthSecretName = "k8s-metrics-service-monitor-auth"
	testOpenshiftMonitoringNamespace           = "openshift-monitoring"
	testCustomerNotificationHTMLPath           = "../templates/customernotification.html"
	testDeploymentType                         = "converged"
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "config", "crd", "bases"),
			filepath.Join("..", "shim", "crds"),
		},
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = ocsv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = promv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = promv1a1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = v1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = opv1a1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = openshiftv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	err = (&ManagedOCSReconciler{
		Client:                       k8sManager.GetClient(),
		UnrestrictedClient:           k8sManager.GetClient(),
		Log:                          ctrl.Log.WithName("controllers").WithName("ManagedOCS"),
		Scheme:                       scheme.Scheme,
		AddonParamSecretName:         testAddonParamsSecretName,
		AddonConfigMapName:           testAddonConfigMapName,
		AddonConfigMapDeleteLabelKey: testAddonConfigMapDeleteLabelKey,
		PagerdutySecretName:          testPagerdutySecretName,
		DeadMansSnitchSecretName:     testDeadMansSnitchSecretName,
		SMTPSecretName:               testSMTPSecretName,
		CustomerNotificationHTMLPath: testCustomerNotificationHTMLPath,
		DeploymentType:               testDeploymentType,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()

	// Client to be use by the test code, using a non cached client
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	ctx := context.Background()

	// Create the primary namespace to be used by some of the tests
	primaryNS := &corev1.Namespace{}
	primaryNS.Name = testPrimaryNamespace
	Expect(k8sClient.Create(ctx, primaryNS)).Should(Succeed())

	// Create a secondary namespace to be used by some of the tests
	secondaryNS := &corev1.Namespace{}
	secondaryNS.Name = testSecondaryNamespace
	Expect(k8sClient.Create(ctx, secondaryNS)).Should(Succeed())

	openshiftMonitoringNS := &corev1.Namespace{}
	openshiftMonitoringNS.Name = testOpenshiftMonitoringNamespace
	Expect(k8sClient.Create(ctx, openshiftMonitoringNS)).Should(Succeed())

	// create a mock deplyer CSV
	deployerCSV := &opv1a1.ClusterServiceVersion{}
	deployerCSV.Name = testDeployerCSVName
	deployerCSV.Namespace = testPrimaryNamespace
	deployerCSV.Spec.InstallStrategy.StrategyName = "test-strategy"
	deployerCSV.Spec.InstallStrategy.StrategySpec.DeploymentSpecs = []opv1a1.StrategyDeploymentSpec{}
	Expect(k8sClient.Create(ctx, deployerCSV)).ShouldNot(HaveOccurred())

	// create a mock OCS CSV
	ocsCSV := &opv1a1.ClusterServiceVersion{}
	ocsCSV.Name = ocsOperatorName
	ocsCSV.Namespace = testPrimaryNamespace
	ocsCSV.Spec.InstallStrategy.StrategyName = "test-strategy"
	ocsCSV.Spec.InstallStrategy.StrategySpec.DeploymentSpecs = getMockOCSCSVDeploymentSpec()
	Expect(k8sClient.Create(ctx, ocsCSV)).ShouldNot(HaveOccurred())

	// Create a mock MCG CSV
	mcgCSV := &opv1a1.ClusterServiceVersion{}
	mcgCSV.Name = mcgOperatorName
	mcgCSV.Namespace = testPrimaryNamespace
	mcgCSV.Spec.InstallStrategy.StrategyName = "test-strategy"
	mcgCSV.Spec.InstallStrategy.StrategySpec.DeploymentSpecs = getMockMCGCSVDeploymentSpec()
	Expect(k8sClient.Create(ctx, mcgCSV)).ShouldNot(HaveOccurred())

	// Create the ManagedOCS resource
	managedOCS := &v1.ManagedOCS{}
	managedOCS.Name = managedOCSName
	managedOCS.Namespace = testPrimaryNamespace
	Expect(k8sClient.Create(ctx, managedOCS)).ShouldNot(HaveOccurred())

	// Create the rook ceph operator config map
	rookConfigMap := &corev1.ConfigMap{}
	rookConfigMap.Name = "rook-ceph-operator-config"
	rookConfigMap.Namespace = testPrimaryNamespace
	rookConfigMap.Data = map[string]string{
		"test-key": "test-value",
	}
	Expect(k8sClient.Create(ctx, rookConfigMap)).ShouldNot(HaveOccurred())

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

func getMockOCSCSVDeploymentSpec() []opv1a1.StrategyDeploymentSpec {
	deploymentSpec := []opv1a1.StrategyDeploymentSpec{
		{
			Name: "deployment1",
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "ocs-operator"},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app": "ocs-operator"},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "ocs-operator",
							},
						},
					},
				},
			},
		},
		{
			Name: "deployment2",
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "rook-ceph-operator"},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app": "rook-ceph-operator"},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "rook-ceph-operator",
							},
						},
					},
				},
			},
		},
		{
			Name: "deployment3",
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "ocs-metrics-exporter"},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app": "ocs-metrics-exporter"},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "ocs-metrics-exporter",
							},
						},
					},
				},
			},
		},
	}
	return deploymentSpec
}

func getMockMCGCSVDeploymentSpec() []opv1a1.StrategyDeploymentSpec {
	deploymentSpec := []opv1a1.StrategyDeploymentSpec{
		{
			Name: "noobaa-operator",
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"noobaa-operator": "deployment"},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"noobaa-operator": "deployment"},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "noobaa-operator",
							},
						},
					},
				},
			},
		},
	}
	return deploymentSpec
}

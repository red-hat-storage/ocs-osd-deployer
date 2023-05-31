package controllers

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	openshiftv1 "github.com/openshift/api/network/v1"
	opv1a1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	ovnv1 "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/egressfirewall/v1"
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	promv1a1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	ocsv1 "github.com/red-hat-storage/ocs-operator/api/v1"
	ocsv1alpha1 "github.com/red-hat-storage/ocs-operator/api/v1alpha1"
	v1 "github.com/red-hat-storage/ocs-osd-deployer/api/v1alpha1"
	"github.com/red-hat-storage/ocs-osd-deployer/templates"
	utils "github.com/red-hat-storage/ocs-osd-deployer/testutils"
	ctrlutils "github.com/red-hat-storage/ocs-osd-deployer/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("ManagedOCS controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		timeout  = time.Second * 3
		interval = time.Millisecond * 250
	)

	ctx := context.Background()
	managedOCSTemplate := &v1.ManagedOCS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      managedOCSName,
			Namespace: testPrimaryNamespace,
		},
	}
	scTemplate := ocsv1.StorageCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      storageClusterName,
			Namespace: testPrimaryNamespace,
		},
	}
	prometheusKubeRBACPoxyConfigMapTemplate := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      templates.PrometheusKubeRBACPoxyConfigMapName,
			Namespace: testPrimaryNamespace,
		},
	}
	prometheusServiceTemplate := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      prometheusServiceName,
			Namespace: testPrimaryNamespace,
		},
	}
	promTemplate := promv1.Prometheus{
		ObjectMeta: metav1.ObjectMeta{
			Name:      prometheusName,
			Namespace: testPrimaryNamespace,
		},
	}
	dmsPromRuleTemplate := promv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dmsRuleName,
			Namespace: testPrimaryNamespace,
		},
	}
	promStsTemplate := appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("prometheus-%s", prometheusName),
			Namespace: testPrimaryNamespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"label": "value"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"label": "value"},
				},
			},
		},
	}
	amTemplate := promv1.Alertmanager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      alertmanagerName,
			Namespace: testPrimaryNamespace,
		},
	}
	amStsTemplate := appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("alertmanager-%s", alertmanagerName),
			Namespace: testPrimaryNamespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"label": "value"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"label": "value"},
				},
			},
		},
	}
	podMonitorTemplate := promv1.PodMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-monitor",
			Namespace: testPrimaryNamespace,
		},
		Spec: promv1.PodMonitorSpec{
			PodMetricsEndpoints: []promv1.PodMetricsEndpoint{},
		},
	}
	serviceMonitorTemplate := promv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service-monitor",
			Namespace: testPrimaryNamespace,
		},
		Spec: promv1.ServiceMonitorSpec{
			Endpoints: []promv1.Endpoint{},
		},
	}
	promRuleTemplate := promv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-prometheus-rule",
			Namespace: testPrimaryNamespace,
		},
	}
	addonParamsSecretTemplate := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testAddonParamsSecretName,
			Namespace: testPrimaryNamespace,
		},
		Data: map[string][]byte{},
	}
	pdSecretTemplate := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testPagerdutySecretName,
			Namespace: testPrimaryNamespace,
		},
		Data: map[string][]byte{},
	}
	dmsSecretTemplate := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDeadMansSnitchSecretName,
			Namespace: testPrimaryNamespace,
		},
		Data: map[string][]byte{},
	}
	smtpSecretTemplate := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSMTPSecretName,
			Namespace: testPrimaryNamespace,
		},
		Data: map[string][]byte{},
	}
	rhobsSecretTemplate := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRHOBSSecretName,
			Namespace: testPrimaryNamespace,
		},
		Data: map[string][]byte{
			"prom-remote-write-config-id":     []byte("test"),
			"prom-remote-write-config-secret": []byte("test"),
			"rhobs-audience":                  []byte("test"),
		},
	}
	amConfigTemplate := promv1a1.AlertmanagerConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      alertmanagerConfigName,
			Namespace: testPrimaryNamespace,
		},
	}
	addonConfigMapTemplate := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testAddonConfigMapName,
			Namespace: testPrimaryNamespace,
		},
	}
	rookConfigMapTemplate := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rook-ceph-operator-config",
			Namespace: testPrimaryNamespace,
		},
		Data: map[string]string{
			"test-key": "test-value",
		},
	}
	egressNetworkPolicyTemplate := openshiftv1.EgressNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      egressNetworkPolicyName,
			Namespace: testPrimaryNamespace,
		},
	}
	egressFirewallTemplate := ovnv1.EgressFirewall{
		ObjectMeta: metav1.ObjectMeta{
			Name:      egressFirewallName,
			Namespace: testPrimaryNamespace,
		},
	}
	ingressNetworkPolicyTemplate := netv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ingressNetworkPolicyName,
			Namespace: testPrimaryNamespace,
		},
	}
	cephIngressNetworkPolicyTemplate := netv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cephIngressNetworkPolicyName,
			Namespace: testPrimaryNamespace,
		},
	}
	providerApiServerIngressNetworkPolicyTemplate := netv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      providerApiServerNetworkPolicyName,
			Namespace: testPrimaryNamespace,
		},
	}
	prometheusProxyIngressNetworkPolicyTemplate := netv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      prometheusProxyNetworkPolicyName,
			Namespace: testPrimaryNamespace,
		},
	}
	pv1StorageClassName := storageClassRbdName
	pv1Template := corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pv-1",
			Namespace: testPrimaryNamespace,
		},
		Spec: corev1.PersistentVolumeSpec{
			StorageClassName: pv1StorageClassName,
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("1Gi"),
			},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				RBD: &corev1.RBDPersistentVolumeSource{
					CephMonitors: []string{
						"0.0.0.0:6789",
					},
					RBDImage: "test",
				},
			},
		},
	}
	pv2StorageClassName := storageClassCephFSName
	pv2Template := corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pv-2",
			Namespace: testSecondaryNamespace,
		},
		Spec: corev1.PersistentVolumeSpec{
			StorageClassName: pv2StorageClassName,
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("1Gi"),
			},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				CephFS: &corev1.CephFSPersistentVolumeSource{
					Monitors: []string{
						"0.0.0.0:6789",
					},
				},
			},
		},
	}
	csvTemplate := opv1a1.ClusterServiceVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDeployerCSVName,
			Namespace: testPrimaryNamespace,
		},
	}
	ocsCSVTemplate := opv1a1.ClusterServiceVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ocsOperatorName,
			Namespace: testPrimaryNamespace,
		},
	}
	ocsInitializationTemplate := ocsv1.OCSInitialization{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ocsinit",
			Namespace: testPrimaryNamespace,
		},
	}
	k8sMetricsServiceMonitorTemplate := promv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sMetricsServiceMonitorName,
			Namespace: testPrimaryNamespace,
		},
	}
	alertRelabelConfigSecretTemplate := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      alertRelabelConfigSecretName,
			Namespace: testPrimaryNamespace,
		},
	}
	mcgCSVTemplate := opv1a1.ClusterServiceVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcgOperatorName,
			Namespace: testPrimaryNamespace,
		},
	}
	onboardingValidationKeySecretTemplate := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      onboardingValidationKeySecretName,
			Namespace: testPrimaryNamespace,
		},
	}
	storageConsumerTemplate := ocsv1alpha1.StorageConsumer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testconsumer",
			Namespace: testPrimaryNamespace,
		},
		Spec: ocsv1alpha1.StorageConsumerSpec{
			Capacity: resource.MustParse("1Gi"),
			Enable:   true,
		},
	}
	odfSubscriptionTemplate := opv1a1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "odf-operator-sub",
			Namespace: testPrimaryNamespace,
		},
	}
	remoteWriteRegexString := "(ALERTS;(CephMdsMissingReplicas|CephMgrIsAbsent|CephMgrIsMissingReplicas|CephNodeDown|" +
		"CephClusterErrorState|CephClusterWarningState|CephOSDVersionMismatch|CephMonVersionMismatch|CephOSDFlapping|" +
		"CephOSDDiskNotResponding|CephOSDDiskUnavailable|CephDataRecoveryTakingTooLong|CephPGRepairTakingTooLong|" +
		"CephMonQuorumAtRisk|CephMonQuorumLost))|(job:ceph_versions_running:count;)|" +
		"(job:ceph_pools_iops_bytes:total;)|(job:ceph_pools_iops:total;)|(job:kube_pv:count;)" +
		"|(job:ceph_osd_metadata:count;)|(ceph_health_status;)|(ceph_cluster_total_used_raw_bytes;)|" +
		"(ceph_cluster_total_bytes;)|(cluster:kubelet_volume_stats_used_bytes:provisioner:sum;)|" +
		"(cluster:kube_persistentvolumeclaim_resource_requests_storage_bytes:provisioner:sum;)"

	setupUninstallConditions := func(
		shouldAddonConfigMapExist bool,
		addonConfigMapDeleteLabel string,
		shouldStorageClusterBeReady bool,
		shouldPrometheusBeReady bool,
		shouldAlertmanagerBeReady bool,
		shouldPV1Exist bool,
		shouldPV2Exist bool,
		shouldStorageConsumersExist bool,
	) {
		// Delete the configmap to ensure that we will not trigger uninstall accidentally
		// via and intermediate state
		configMap := addonConfigMapTemplate.DeepCopy()
		err := k8sClient.Delete(ctx, configMap)
		Expect(err == nil || errors.IsNotFound(err)).Should(BeTrue())

		// Setup storagecluster state
		sc := scTemplate.DeepCopy()
		Expect(k8sClient.Get(ctx, utils.GetResourceKey(sc), sc)).Should(Succeed())
		if shouldStorageClusterBeReady {
			sc.Status.Phase = "Ready"
		} else {
			sc.Status.Phase = ""
		}
		Expect(k8sClient.Status().Update(ctx, sc)).Should(Succeed())

		// Setup prometheus state
		prom := promTemplate.DeepCopy()
		Expect(k8sClient.Get(ctx, utils.GetResourceKey(prom), prom)).Should(Succeed())
		desiredReplicas := int32(1)
		if prom.Spec.Replicas != nil {
			desiredReplicas = *prom.Spec.Replicas
		}
		promSts := promStsTemplate.DeepCopy()
		Expect(k8sClient.Get(ctx, utils.GetResourceKey(promSts), promSts)).Should(Succeed())
		if shouldPrometheusBeReady {
			promSts.Status.Replicas = desiredReplicas
			promSts.Status.ReadyReplicas = desiredReplicas
		} else {
			promSts.Status.Replicas = 0
			promSts.Status.ReadyReplicas = 0
		}
		Expect(k8sClient.Status().Update(ctx, promSts)).Should(Succeed())

		// Setup alertmanager state
		am := amTemplate.DeepCopy()
		Expect(k8sClient.Get(ctx, utils.GetResourceKey(am), am)).Should(Succeed())
		desiredReplicas = int32(1)
		if am.Spec.Replicas != nil {
			desiredReplicas = *am.Spec.Replicas
		}
		amSts := amStsTemplate.DeepCopy()
		Expect(k8sClient.Get(ctx, utils.GetResourceKey(amSts), amSts)).Should(Succeed())
		if shouldAlertmanagerBeReady {
			amSts.Status.Replicas = desiredReplicas
			amSts.Status.ReadyReplicas = desiredReplicas
		} else {
			amSts.Status.Replicas = 0
			amSts.Status.ReadyReplicas = 0
		}
		Expect(k8sClient.Status().Update(ctx, amSts)).Should(Succeed())

		// Setup pv1 state (an rbd backed pv in the primary namespace)
		pv1 := pv1Template.DeepCopy()
		if shouldPV1Exist {
			err := k8sClient.Create(ctx, pv1)
			Expect(err == nil || errors.IsAlreadyExists(err)).Should(BeTrue())
		} else {
			err := k8sClient.Get(ctx, utils.GetResourceKey(pv1), pv1)
			if err == nil {
				Expect(k8sClient.Delete(ctx, pv1)).Should(Succeed())
			} else {
				Expect(errors.IsNotFound(err)).Should(BeTrue())
			}
		}

		// Setup pv2 state (an cephfs backed pv in the secondary namespace)
		pv2 := pv2Template.DeepCopy()
		if shouldPV2Exist {
			err := k8sClient.Create(ctx, pv2)
			Expect(err == nil || errors.IsAlreadyExists(err)).Should(BeTrue())
		} else {
			err := k8sClient.Get(ctx, utils.GetResourceKey(pv2), pv2)
			if err == nil {
				Expect(k8sClient.Delete(ctx, pv2)).Should(Succeed())
			} else {
				Expect(errors.IsNotFound(err)).Should(BeTrue())
			}
		}

		// setup add-on configmap state
		if shouldAddonConfigMapExist {
			labels := map[string]string{}
			if addonConfigMapDeleteLabel != "" {
				labels[addonConfigMapDeleteLabel] = "dummy"
			}
			configMap.SetLabels(labels)
			Expect(k8sClient.Create(ctx, configMap)).Should(Succeed())
		}
		cons := storageConsumerTemplate.DeepCopy()
		if shouldStorageConsumersExist {
			Expect(k8sClient.Create(ctx, cons)).Should(Succeed())
		} else {
			err := k8sClient.Get(ctx, utils.GetResourceKey(cons), cons)
			if err == nil {
				Expect(k8sClient.Delete(ctx, cons)).Should(Succeed())
			} else {
				Expect(errors.IsNotFound(err)).Should(BeTrue())
			}
		}
	}

	setupAlertmanagerConfigConditions := func(
		shouldPagerdutySecretExist bool,
		hasPagerdutyKey bool,
		shouldDMSSecretExist bool,
		hasSnitchUrl bool,
		shouldSMTPSecretExist bool,
		hasValueForSMTP bool,
	) {
		smtpSecret := smtpSecretTemplate.DeepCopy()
		var smtpSecretExists bool
		if err := k8sClient.Get(ctx, utils.GetResourceKey(smtpSecret), smtpSecret); err == nil {
			smtpSecretExists = true
		} else if errors.IsNotFound(err) {
			smtpSecretExists = false
		} else {
			Expect(err).ToNot(HaveOccurred())
		}
		if shouldSMTPSecretExist {
			if hasValueForSMTP {
				smtpSecret.Data["host"] = []byte("test-host")
				smtpSecret.Data["port"] = []byte("20")
				smtpSecret.Data["username"] = []byte("test-key")
				smtpSecret.Data["password"] = []byte("test-password")
			} else {
				smtpSecret.Data["host"] = []byte("")
				smtpSecret.Data["port"] = []byte("")
				smtpSecret.Data["username"] = []byte("")
				smtpSecret.Data["password"] = []byte("")
			}
			if smtpSecretExists {
				Expect(k8sClient.Update(ctx, smtpSecret)).Should(Succeed())
			} else {
				Expect(k8sClient.Create(ctx, smtpSecret)).Should(Succeed())
			}
		} else if smtpSecretExists {
			Expect(k8sClient.Delete(ctx, smtpSecret)).Should(Succeed())
		}

		dmsSecret := dmsSecretTemplate.DeepCopy()
		var dmsSecretExists bool
		if err := k8sClient.Get(ctx, utils.GetResourceKey(dmsSecret), dmsSecret); err == nil {
			dmsSecretExists = true
		} else if errors.IsNotFound(err) {
			dmsSecretExists = false
		} else {
			Expect(err).ToNot(HaveOccurred())
		}
		if shouldDMSSecretExist {
			if hasSnitchUrl {
				dmsSecret.Data["SNITCH_URL"] = []byte("https://nosnch.in/fake_url")
			} else {
				dmsSecret.Data["SNITCH_URL"] = []byte("")
			}
			if dmsSecretExists {
				Expect(k8sClient.Update(ctx, dmsSecret)).Should(Succeed())
			} else {
				Expect(k8sClient.Create(ctx, dmsSecret)).Should(Succeed())
			}
		} else if dmsSecretExists {
			Expect(k8sClient.Delete(ctx, dmsSecret)).Should(Succeed())
		}

		pdSecret := pdSecretTemplate.DeepCopy()
		var pdSecretExists bool
		if err := k8sClient.Get(ctx, utils.GetResourceKey(pdSecret), pdSecret); err == nil {
			pdSecretExists = true
		} else if errors.IsNotFound(err) {
			pdSecretExists = false
		} else {
			Expect(err).ToNot(HaveOccurred())
		}
		if shouldPagerdutySecretExist {
			if hasPagerdutyKey {
				pdSecret.Data["PAGERDUTY_KEY"] = []byte("test-key")
			} else {
				pdSecret.Data["PAGERDUTY_KEY"] = []byte("")
			}
			if pdSecretExists {
				Expect(k8sClient.Update(ctx, pdSecret)).Should(Succeed())
			} else {
				Expect(k8sClient.Create(ctx, pdSecret)).Should(Succeed())
			}
		} else if pdSecretExists {
			Expect(k8sClient.Delete(ctx, pdSecret)).Should(Succeed())
		}
	}

	Context("reconcile()", Ordered, func() {
		When("there is no add-on parameters secret in the cluster", func() {
			It("should not create a reconciled resources", func() {
				// Verify that a secret is not present
				secret := addonParamsSecretTemplate.DeepCopy()
				Expect(k8sClient.Get(ctx, utils.GetResourceKey(secret), secret)).Should(
					WithTransform(errors.IsNotFound, BeTrue()),
				)

				//Ensure, over a period of time, that the resources are not created
				resList := []client.Object{
					scTemplate.DeepCopy(),
					promTemplate.DeepCopy(),
					amTemplate.DeepCopy(),
				}
				utils.EnsureNoResources(k8sClient, ctx, resList, timeout, interval)
			})
		})
		When("there is no size field in the add-on parameters secret", func() {
			It("should not create a reconciled resources", func() {
				// quota size isn't enforced by provider when consumer supplies it
				if testReconciler.DeploymentType == consumerDeploymentType {
					Skip(fmt.Sprintf("Skipping the test as it is not required by %v", testReconciler.DeploymentType))
				}

				// Create empty add-on parameters secret
				secret := addonParamsSecretTemplate.DeepCopy()
				Expect(k8sClient.Create(ctx, secret)).Should(Succeed())

				// Ensure, over a period of time, that the resources are not created
				resList := []client.Object{
					scTemplate.DeepCopy(),
					promTemplate.DeepCopy(),
					amTemplate.DeepCopy(),
				}
				utils.EnsureNoResources(k8sClient, ctx, resList, timeout, interval)

				// Remove the secret for future cases
				Expect(k8sClient.Delete(ctx, secret)).Should(Succeed())
			})
		})
		When("there is an invalid size value in the add-on parameters secret", func() {
			It("should not create reconciled resources", func() {
				// quota size isn't enforced by provider when consumer supplies it
				if testReconciler.DeploymentType == consumerDeploymentType {
					Skip(fmt.Sprintf("Skipping the test as it is not required by %v", testReconciler.DeploymentType))
				}

				// Create a invalid add-on parameters secret
				secret := addonParamsSecretTemplate.DeepCopy()
				secret.Data["size"] = []byte("AA")
				Expect(k8sClient.Create(ctx, secret)).Should(Succeed())

				// Ensure, over a period of time, that the resources are not created
				resList := []client.Object{
					scTemplate.DeepCopy(),
					promTemplate.DeepCopy(),
					amTemplate.DeepCopy(),
				}
				utils.EnsureNoResources(k8sClient, ctx, resList, timeout, interval)

				// Remove the secret for future cases
				Expect(k8sClient.Delete(ctx, secret)).Should(Succeed())
			})
		})
		When("there is no enable-mcg field in the add-on parameters secret", func() {
			It("should not create storagecluster", func() {
				Skip("Skipping this test till Nooba MCG is integrated in deployer")
				secret := addonParamsSecretTemplate.DeepCopy()
				secret.Data["size"] = []byte("4")
				Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
				// Ensure, over a period of time, that the resource is not created
				utils.EnsureNoResource(k8sClient, ctx, scTemplate.DeepCopy(), timeout, interval)
			})
		})
		When("there is an invalid enable-mcg value in the add-on parameters secret", func() {
			It("should not create storagecluster", func() {
				// Create a invalid enable-mcg parameter value
				secret := addonParamsSecretTemplate.DeepCopy()
				secret.Data["size"] = []byte("4")
				secret.Data["enable-mcg"] = []byte("NO")
				Expect(k8sClient.Create(ctx, secret)).Should(Succeed())

				// Ensure, over a period of time, that the resource is not created
				utils.EnsureNoResource(k8sClient, ctx, scTemplate.DeepCopy(), timeout, interval)
			})
		})
		When("there is a valid enable-mcg value in the addon-on parameters secret which is false", func() {
			It("should not change storagecluster's MCG reconcile strategy to manage", func() {
				secret := addonParamsSecretTemplate.DeepCopy()
				secret.Data["size"] = []byte("4")
				secret.Data["enable-mcg"] = []byte("false")
				Expect(k8sClient.Update(ctx, secret)).Should(Succeed())

				Eventually(func() bool {
					sc := scTemplate.DeepCopy()
					err := k8sClient.Get(ctx, utils.GetResourceKey(sc), sc)
					if err != nil {
						return true
					}
					return sc.Spec.MultiCloudGateway.ReconcileStrategy == "manage"
				}, timeout, interval).Should(BeFalse())
			})
		})
		When("there is a valid enable-mcg value in the addon-on parameters secret which is true", func() {
			It("should change storagecluster's MCG reconcile strategy to manage", func() {
				secret := addonParamsSecretTemplate.DeepCopy()
				secret.Data["size"] = []byte("4")
				secret.Data["enable-mcg"] = []byte("true")
				Expect(k8sClient.Update(ctx, secret)).Should(Succeed())

				Eventually(func() bool {
					sc := scTemplate.DeepCopy()
					err := k8sClient.Get(ctx, utils.GetResourceKey(sc), sc)
					if err != nil {
						return false
					}
					return sc.Spec.MultiCloudGateway.ReconcileStrategy == "manage"
				}, timeout, interval).Should(BeTrue())
			})
		})
		When("there is no rhobs secret", func() {
			It("should  create prometheus resource without remote write spec", func() {
				if testReconciler.DeploymentType != convergedDeploymentType {
					utils.WaitForResource(k8sClient, ctx, promTemplate.DeepCopy(), timeout, interval)
					Eventually(func() bool {
						prom := promTemplate.DeepCopy()
						err := k8sClient.Get(ctx, utils.GetResourceKey(prom), prom)
						if err != nil {
							return false
						}
						return len(prom.Spec.RemoteWrite) == 0
					}, timeout, interval).Should(BeTrue())
				}
			})
		})
		When("there is a rhobs secret with missing values", func() {
			It("should not create prometheus resource with remote write spec", func() {
				if testReconciler.DeploymentType != convergedDeploymentType {
					secret := rhobsSecretTemplate.DeepCopy()
					delete(secret.Data, "prom-remote-write-config-id")
					Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
					Eventually(func() bool {
						prom := promTemplate.DeepCopy()
						err := k8sClient.Get(ctx, utils.GetResourceKey(prom), prom)
						if err != nil {
							return false
						}
						return len(prom.Spec.RemoteWrite) == 0
					}, timeout, interval).Should(BeTrue())
				}
			})
		})
		When("there is a rhobs secret", func() {
			It("should create prometheus resources", func() {
				if testReconciler.DeploymentType != convergedDeploymentType {
					secret := rhobsSecretTemplate.DeepCopy()
					Expect(k8sClient.Update(ctx, secret)).Should(Succeed())
					utils.WaitForResource(k8sClient, ctx, promTemplate.DeepCopy(), timeout, interval)
					Eventually(func() bool {
						prom := promTemplate.DeepCopy()
						err := k8sClient.Get(ctx, utils.GetResourceKey(prom), prom)
						if err != nil {
							return false
						}
						return len(prom.Spec.RemoteWrite) > 0
					}, timeout, interval).Should(BeTrue())
				}
			})
			It("should have approriate remote-write relabel configs", func() {
				if testReconciler.DeploymentType != convergedDeploymentType {
					prom := promTemplate.DeepCopy()
					utils.WaitForResource(k8sClient, ctx, prom, timeout, interval)
					Expect(prom.Spec.RemoteWrite[0].WriteRelabelConfigs[0].SourceLabels).Should(Equal([]string{"__name__", "alertname"}))
					Expect(prom.Spec.RemoteWrite[0].WriteRelabelConfigs[0].Regex).Should(Equal(remoteWriteRegexString))
					Expect(prom.Spec.RemoteWrite[0].WriteRelabelConfigs[0].Action).Should(Equal("keep"))
				}
			})
		})
		When("MCG is already enabled and enable-mcg value in addon-on parameter secret is false", func() {
			It("should not change storagecluster's MCG reconcile strategy to 'ignore' or 'standalone'", func() {
				secret := addonParamsSecretTemplate.DeepCopy()
				secret.Data["enable-mcg"] = []byte("false")
				Expect(k8sClient.Update(ctx, secret)).Should(Succeed())
				// MCG spec should not get updated
				Eventually(func() bool {
					sc := scTemplate.DeepCopy()
					err := k8sClient.Get(ctx, utils.GetResourceKey(sc), sc)
					if err != nil {
						return false
					}
					return sc.Spec.MultiCloudGateway.ReconcileStrategy == "manage"
				}, timeout, interval).Should(BeTrue())
				// Remove the secret for future cases
				Expect(k8sClient.Delete(ctx, secret)).Should(Succeed())
				// Delete created resources
				resList := []client.Object{
					scTemplate.DeepCopy(),
					promTemplate.DeepCopy(),
					amTemplate.DeepCopy(),
				}
				for _, object := range resList {
					Expect(k8sClient.Delete(ctx, object)).Should(Succeed())
				}
			})
		})
		When("there is a valid size in the add-on parameter secret", func() {
			It("should create reconciled resources", func() {

				// Create a valid add-on parameters secret
				secret := addonParamsSecretTemplate.DeepCopy()
				switch testReconciler.DeploymentType {
				case convergedDeploymentType:
					secret.Data["size"] = []byte("1")
					secret.Data["enable-mcg"] = []byte("false")
				case consumerDeploymentType:
					// "size" and "unit" params are depreacted for consumer deployment
					secret.Data["enable-mcg"] = []byte("false")
					secret.Data["onboarding-ticket"] = []byte("onboarding-tickets")
					secret.Data["storage-provider-endpoint"] = []byte("0.0.0.0:36179")
				case providerDeploymentType:
					secret.Data["size"] = []byte("4")
					secret.Data["enable-mcg"] = []byte("false")
					secret.Data["onboarding-validation-key"] = []byte("   test-validation-key   ")
				}
				Expect(k8sClient.Create(ctx, secret)).Should(Succeed())

				if testReconciler.DeploymentType == providerDeploymentType {
					By("Creating a onboarding validation key secret resource")
					utils.WaitForResource(k8sClient, ctx, onboardingValidationKeySecretTemplate.DeepCopy(), timeout, interval)
				}

				By("Creating a storagecluster resource")
				sc := scTemplate.DeepCopy()
				utils.WaitForResource(k8sClient, ctx, sc, timeout, interval)
				switch testReconciler.DeploymentType {
				case convergedDeploymentType:
					ds := sc.Spec.StorageDeviceSets[0]
					Expect(ds.Count).Should(Equal(1))
				case consumerDeploymentType:
					es := &sc.Spec.ExternalStorage
					Expect(es.Enable).Should(BeTrue())
					Expect(es.OnboardingTicket).Should(Equal("onboarding-tickets"))
					Expect(es.StorageProviderEndpoint).Should(Equal("0.0.0.0:36179"))

					want, _ := resource.ParseQuantity("1024Ti")
					Expect(es.RequestedCapacity.Cmp(want)).Should(Equal(0))
				case providerDeploymentType:
					ds := sc.Spec.StorageDeviceSets[0]
					Expect(ds.Count).Should(Equal(1))
					Expect(sc.Spec.HostNetwork).Should(BeTrue())
					Expect(sc.Spec.AllowRemoteStorageConsumers).Should(BeTrue())
				}

				By("Creating a kube-rbac-proxy configmap")
				utils.WaitForResource(k8sClient, ctx, prometheusKubeRBACPoxyConfigMapTemplate.DeepCopy(), timeout, interval)

				By("Creating a prometheus service resource")
				utils.WaitForResource(k8sClient, ctx, prometheusServiceTemplate.DeepCopy(), timeout, interval)

				By("Creating a prometheus resource")
				utils.WaitForResource(k8sClient, ctx, promTemplate.DeepCopy(), timeout, interval)

				By("Creating an alertmanager resource")
				utils.WaitForResource(k8sClient, ctx, amTemplate.DeepCopy(), timeout, interval)
			})
		})
		When("onboarding validation data was passed to addon param secret", func() {
			It("should trim whitespace and add header and footer to the data", func() {
				if testReconciler.DeploymentType != providerDeploymentType {
					Skip(fmt.Sprintf("Skipping the test as it is not required by %v", testReconciler.DeploymentType))
				}
				secret := onboardingValidationKeySecretTemplate.DeepCopy()
				utils.WaitForResource(k8sClient, ctx, secret, timeout, interval)
				Expect(secret.Data["key"]).Should(Equal([]byte("-----BEGIN PUBLIC KEY-----\ntest-validation-key\n-----END PUBLIC KEY-----")))
			})
		})
		When("size is increased in the add-on parameters secret", func() {
			It("should increase storagecluster's storage device set count", func() {
				if testReconciler.DeploymentType == consumerDeploymentType {
					Skip(fmt.Sprintf("Skipping the test as it is not required by %v", testReconciler.DeploymentType))
				}
				secret := addonParamsSecretTemplate.DeepCopy()
				switch testReconciler.DeploymentType {
				case convergedDeploymentType:
					secret.Data["size"] = []byte("2")
					secret.Data["enable-mcg"] = []byte("false")
				case providerDeploymentType:
					secret.Data["size"] = []byte("8")
					secret.Data["enable-mcg"] = []byte("false")
				}
				Expect(k8sClient.Update(ctx, secret)).Should(Succeed())

				// wait for the storagecluster to update
				Eventually(func() bool {
					sc := scTemplate.DeepCopy()
					err := k8sClient.Get(ctx, utils.GetResourceKey(sc), sc)
					if err != nil {
						return false
					}

					var ds *ocsv1.StorageDeviceSet = nil
					for index := range sc.Spec.StorageDeviceSets {
						item := &sc.Spec.StorageDeviceSets[index]
						if item.Name == deviceSetName {
							ds = item
							break
						}
					}
					return ds != nil && ds.Count == 2
				}, timeout, interval).Should(BeTrue())

			})
		})
		When("size is decreased in the add-on parameters secret", func() {
			It("should not decrease storagecluster's storage device set count", func() {
				if testReconciler.DeploymentType == consumerDeploymentType {
					Skip(fmt.Sprintf("Skipping the test as it is not required by %v", testReconciler.DeploymentType))
				}

				secret := addonParamsSecretTemplate.DeepCopy()
				switch testReconciler.DeploymentType {
				case convergedDeploymentType:
					secret.Data["size"] = []byte("1")
					secret.Data["enable-mcg"] = []byte("false")
				case providerDeploymentType:
					secret.Data["size"] = []byte("4")
					secret.Data["enable-mcg"] = []byte("false")
				}
				Expect(k8sClient.Update(ctx, secret)).Should(Succeed())

				Consistently(func() bool {
					sc := scTemplate.DeepCopy()
					err := k8sClient.Get(ctx, utils.GetResourceKey(sc), sc)
					if err != nil {
						return false
					}

					var ds *ocsv1.StorageDeviceSet = nil
					for index := range sc.Spec.StorageDeviceSets {
						item := &sc.Spec.StorageDeviceSets[index]
						if item.Name == deviceSetName {
							ds = item
							break
						}
					}
					return ds != nil && ds.Count == 2
				}, timeout, interval).Should(BeTrue())

				// Revert the size in add-on param secret
				switch testReconciler.DeploymentType {
				case convergedDeploymentType:
					secret.Data["size"] = []byte("2")
				case providerDeploymentType:
					secret.Data["size"] = []byte("8")
				}
				Expect(k8sClient.Update(ctx, secret)).Should(Succeed())
			})
		})
		When("there is a rook-ceph-operator-config ConfigMap", func() {
			It("should ensure there are RBD CSI resource limits", func() {
				configMap := rookConfigMapTemplate.DeepCopy()
				Eventually(func() bool {
					err := k8sClient.Get(ctx, utils.GetResourceKey(configMap), configMap)
					if err != nil {
						return false
					}

					if testReconciler.DeploymentType == providerDeploymentType {
						return configMap.Data["ROOK_CSI_ENABLE_CEPHFS"] == "false" &&
							configMap.Data["ROOK_CSI_ENABLE_RBD"] == "false"
					} else {
						return configMap.Data["CSI_RBD_PROVISIONER_RESOURCE"] != "" &&
							configMap.Data["CSI_RBD_PLUGIN_RESOURCE"] != "" &&
							configMap.Data["CSI_CEPHFS_PROVISIONER_RESOURCE"] != "" &&
							configMap.Data["CSI_CEPHFS_PLUGIN_RESOURCE"] != ""
					}
				}, timeout, interval).Should(BeTrue())
			})

			It("should not modify unrelated configurations", func() {
				configMap := rookConfigMapTemplate.DeepCopy()
				Expect(k8sClient.Get(ctx, utils.GetResourceKey(configMap), configMap)).Should(Succeed())
				Expect(configMap.Data["test-key"]).Should(Equal("test-value"))
			})
		})
		When("current odf subscription channel is different from the desired subscription channel", func() {
			It("should update the odf subscription channel", func() {
				Eventually(func() string {
					testSubscription := odfSubscriptionTemplate.DeepCopy()
					Expect(k8sClient.Get(ctx, utils.GetResourceKey(testSubscription), testSubscription)).Should(Succeed())
					return testSubscription.Spec.Channel
				}, timeout, interval).Should(Equal(desiredODFSubscriptionChannel))
			})
		})
		When("there is no rook-ceph-operator-config ConfigMap", func() {
			It("should not create reconciled resources", func() {
				resList := []client.Object{
					scTemplate.DeepCopy(),
					promTemplate.DeepCopy(),
					amTemplate.DeepCopy(),
				}

				// Delete the configMap which will throw an error that
				// the rook-ceph-operator-config configMap not found
				// This will block creation of resources that gets created
				// after reconcile of rook-ceph-operator-config configMap
				configMap := rookConfigMapTemplate.DeepCopy()
				Expect(k8sClient.Delete(ctx, configMap)).Should(Succeed())

				// Delete the resource as they already exist because reconcile
				// was working fine before deleting rook-ceph-operator-config configMap
				for _, object := range resList {
					Expect(k8sClient.Delete(ctx, object)).Should(Succeed())
				}

				// Resource shouldn't get created as the rook-ceph-operator-config configMap
				// was deleted
				utils.EnsureNoResources(k8sClient, ctx, resList, timeout, interval)

				// Recreate the configMap for other tests.
				Expect(k8sClient.Create(ctx, configMap)).Should(Succeed())

				// Ensure resources get created again so that other tests work properly
				for _, object := range resList {
					utils.WaitForResource(k8sClient, ctx, object, timeout, interval)
				}
			})
		})
		When("the storagecluster is not ready", func() {
			BeforeEach(func() {
				// Ensure that the storagecluster is not ready
				// This test assumes a StorageCluster is already created.
				sc := scTemplate.DeepCopy()
				Expect(k8sClient.Get(ctx, utils.GetResourceKey(sc), sc)).Should(Succeed())

				// Updating the Status of the StorageCluster should trigger a reconcile for managed-ocs
				sc.Status.Phase = "Pending"
				Expect(k8sClient.Status().Update(ctx, sc)).Should(Succeed())
			})
			It("should reflect that in the ManagedOCS resource status", func() {
				By("by setting Status.Components.StorageCluster.State to Pending")
				Eventually(func() v1.ComponentState {
					managedOCS := managedOCSTemplate.DeepCopy()
					Expect(k8sClient.Get(ctx, utils.GetResourceKey(managedOCS), managedOCS)).Should(Succeed())
					return managedOCS.Status.Components.StorageCluster.State
				}, timeout, interval).Should(Equal(v1.ComponentPending))
			})
		})
		When("the storeagecluster is ready", func() {
			BeforeEach(func() {
				// Ensure that the storagecluster is ready
				// This test assumes a StorageCluster is already created.
				sc := scTemplate.DeepCopy()
				Expect(k8sClient.Get(ctx, utils.GetResourceKey(sc), sc)).Should(Succeed())

				// Updating the Status of the storagecluster resource should trigger a reconcile for managedocs
				sc.Status.Phase = "Ready"
				Expect(k8sClient.Status().Update(ctx, sc)).Should(Succeed())
			})
			It("should refelct that in the ManagedOCS resource status", func() {
				By("by setting Status.Components.StorageCluster.State to Ready")
				Eventually(func() v1.ComponentState {
					managedOCS := managedOCSTemplate.DeepCopy()
					Expect(k8sClient.Get(ctx, utils.GetResourceKey(managedOCS), managedOCS)).Should(Succeed())
					return managedOCS.Status.Components.StorageCluster.State
				}, timeout, interval).Should(Equal(v1.ComponentReady))
			})
		})
		When("the alertRelabelConfigSecret resource is deleted", func() {
			It("should create a new alertRelabelConfigSecret in the namespace", func() {
				// Delete the alertRelabelConfigSecret resource
				Expect(k8sClient.Delete(ctx, alertRelabelConfigSecretTemplate.DeepCopy())).Should(Succeed())

				// Wait for the alertRelabelConfigSecret to be recreated
				utils.WaitForResource(k8sClient, ctx, alertRelabelConfigSecretTemplate.DeepCopy(), timeout, interval)
			})
		})
		When("the kube-rbac-proxy ConfigMap is modified", func() {
			It("should revert the changes and bring the resource back to its managed state", func() {
				// Get an updated configMap
				kubeRBACConfigMap := prometheusKubeRBACPoxyConfigMapTemplate.DeepCopy()
				kubeRBACConfigMapKey := utils.GetResourceKey(kubeRBACConfigMap)
				Expect(k8sClient.Get(ctx, kubeRBACConfigMapKey, kubeRBACConfigMap)).Should(Succeed())

				// Update to the configMap with dummy values
				data := kubeRBACConfigMap.Data
				kubeRBACConfigMap.Data = map[string]string{}
				Expect(k8sClient.Update(ctx, kubeRBACConfigMap)).Should(Succeed())

				// Wait for the configMap changes to be reverted
				Eventually(func() map[string]string {
					kubeRBACConfigMap := prometheusKubeRBACPoxyConfigMapTemplate.DeepCopy()
					Expect(k8sClient.Get(ctx, kubeRBACConfigMapKey, kubeRBACConfigMap)).Should(Succeed())
					return kubeRBACConfigMap.Data
				}, timeout, interval).Should(Equal(data))
			})
		})
		When("the kube-rbac-proxy ConfigMap resource is deleted", func() {
			It("should create a new kube-rbac-proxy ConfigMap in the namespace", func() {
				// Delete the prometheusKubeRBACPoxyConfigMap resource
				Expect(k8sClient.Delete(ctx, prometheusKubeRBACPoxyConfigMapTemplate.DeepCopy())).Should(Succeed())

				// Wait for the prometheusKubeRBACPoxyConfigMap to be recreated
				utils.WaitForResource(k8sClient, ctx, prometheusKubeRBACPoxyConfigMapTemplate.DeepCopy(), timeout, interval)
			})
		})
		When("the prometheus service is modified", func() {
			It("should revert the changes and bring the resource back to its managed state", func() {
				// Get an updated prometheus service
				prometheusService := prometheusServiceTemplate.DeepCopy()
				prometheusServiceKey := utils.GetResourceKey(prometheusService)
				Expect(k8sClient.Get(ctx, prometheusServiceKey, prometheusService)).Should(Succeed())

				// Update to empty spec
				spec := prometheusService.Spec
				// Added dummy value because ports is a required field
				// and clusterIP is an immutable field for service resource
				prometheusService.Spec = corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Port: 1234,
						},
					},
					ClusterIP: spec.ClusterIP,
				}
				Expect(k8sClient.Update(ctx, prometheusService)).Should(Succeed())

				// Wait for the spec changes to be reverted
				Eventually(func() corev1.ServiceSpec {
					prometheusService := prometheusService.DeepCopy()
					Expect(k8sClient.Get(ctx, prometheusServiceKey, prometheusService)).Should(Succeed())
					return prometheusService.Spec
				}, timeout, interval).Should(Equal(spec))
			})
		})
		When("the prometheus service resource is deleted", func() {
			It("should create a new prometheus service in the namespace", func() {
				// Delete the prometheus service resource
				Expect(k8sClient.Delete(ctx, prometheusServiceTemplate.DeepCopy())).Should(Succeed())

				// Wait for the prometheus service to be recreated
				utils.WaitForResource(k8sClient, ctx, prometheusServiceTemplate.DeepCopy(), timeout, interval)
			})
		})
		When("prometheus has non-ready replicas", func() {
			It("should reflect that in the ManagedOCS resource status", func() {
				By("by setting Status.Components.Prometheus.State to Pending")
				// Updating the status of the prometheus statefulset should trigger a reconcile for managedocs
				promSts := promStsTemplate.DeepCopy()
				Expect(k8sClient.Create(ctx, promSts)).Should(Succeed())

				promSts.Status.Replicas = 0
				promSts.Status.ReadyReplicas = 0
				Expect(k8sClient.Status().Update(ctx, promSts)).Should(Succeed())

				managedOCS := managedOCSTemplate.DeepCopy()
				key := utils.GetResourceKey(managedOCS)
				Eventually(func() v1.ComponentState {
					Expect(k8sClient.Get(ctx, key, managedOCS)).Should(Succeed())
					return managedOCS.Status.Components.Prometheus.State
				}, timeout, interval).Should(Equal(v1.ComponentPending))
			})
		})
		When("all prometheus replicas are ready", func() {
			It("should reflect that in the ManagedOCS resource status", func() {
				By("by setting Status.Components.Prometheus.State to Ready")
				prom := promTemplate.DeepCopy()
				Expect(k8sClient.Get(ctx, utils.GetResourceKey(prom), prom)).Should(Succeed())
				desiredReplicas := int32(1)
				if prom.Spec.Replicas != nil {
					desiredReplicas = *prom.Spec.Replicas
				}

				// Updating the status of the prometheus statefulset should trigger a reconcile for managedocs
				promSts := promStsTemplate.DeepCopy()
				promSts.Status.Replicas = desiredReplicas
				promSts.Status.ReadyReplicas = desiredReplicas
				Expect(k8sClient.Status().Update(ctx, promSts)).Should(Succeed())

				managedOCS := managedOCSTemplate.DeepCopy()
				key := utils.GetResourceKey(managedOCS)
				Eventually(func() v1.ComponentState {
					Expect(k8sClient.Get(ctx, key, managedOCS)).Should(Succeed())
					return managedOCS.Status.Components.Prometheus.State
				}, timeout, interval).Should(Equal(v1.ComponentReady))
			})
		})
		When("alertmanager has non-ready replicas", func() {
			It("should reflect that in the ManagedOCS resource status", func() {
				By("by setting Status.Components.Alertmanager.State to Pending")
				// Updating the status of the alertmanager statefulset should trigger a reconcile for managedocs
				amSts := amStsTemplate.DeepCopy()
				Expect(k8sClient.Create(ctx, amSts)).Should(Succeed())

				amSts.Status.Replicas = 0
				amSts.Status.ReadyReplicas = 0
				Expect(k8sClient.Status().Update(ctx, amSts)).Should(Succeed())

				managedOCS := managedOCSTemplate.DeepCopy()
				key := utils.GetResourceKey(managedOCS)
				Eventually(func() v1.ComponentState {
					Expect(k8sClient.Get(ctx, key, managedOCS)).Should(Succeed())
					return managedOCS.Status.Components.Alertmanager.State
				}, timeout, interval).Should(Equal(v1.ComponentPending))
			})
		})
		When("all alertmanager replicas are ready", func() {
			It("should reflect that in the ManagedOCS resource status", func() {
				By("by setting Status.Components.Alertmanager.State to Ready")
				am := amTemplate.DeepCopy()
				Expect(k8sClient.Get(ctx, utils.GetResourceKey(am), am)).Should(Succeed())
				desiredReplicas := int32(1)
				if am.Spec.Replicas != nil {
					desiredReplicas = *am.Spec.Replicas
				}

				// Updating the status of the alertmanager statefulset should trigger a reconcile for managedocs
				amSts := amStsTemplate.DeepCopy()
				amSts.Status.Replicas = desiredReplicas
				amSts.Status.ReadyReplicas = desiredReplicas
				Expect(k8sClient.Status().Update(ctx, amSts)).Should(Succeed())

				managedOCS := managedOCSTemplate.DeepCopy()
				key := utils.GetResourceKey(managedOCS)
				Eventually(func() v1.ComponentState {
					Expect(k8sClient.Get(ctx, key, managedOCS)).Should(Succeed())
					return managedOCS.Status.Components.Alertmanager.State
				}, timeout, interval).Should(Equal(v1.ComponentReady))
			})
		})
		When("the storagecluster resource is deleted", func() {
			It("should create a new storagecluster in the namespace", func() {
				// Delete the storagecluster resource
				Expect(k8sClient.Delete(ctx, scTemplate.DeepCopy())).Should(Succeed())

				// Wait for the storagecluster to be recreated
				utils.WaitForResource(k8sClient, ctx, scTemplate.DeepCopy(), timeout, interval)
			})
		})
		When("the storagecluster resource is modified while the reconcile strategy is not set", func() {
			It("should revert the changes and bring the resource back to its managed state", func() {
				// Set managed OCS to reconcile strategy to strict
				managedOCS := managedOCSTemplate.DeepCopy()
				Expect(k8sClient.Get(ctx, utils.GetResourceKey(managedOCS), managedOCS)).Should(Succeed())
				managedOCS.Spec.ReconcileStrategy = ""
				Expect(k8sClient.Update(ctx, managedOCS)).Should(Succeed())

				// Get an updated storagecluster
				sc := scTemplate.DeepCopy()
				scKey := utils.GetResourceKey(sc)
				Expect(k8sClient.Get(ctx, scKey, sc)).Should(Succeed())

				// Update to empty spec
				spec := sc.Spec.DeepCopy()
				sc.Spec = ocsv1.StorageClusterSpec{}
				Expect(k8sClient.Update(ctx, sc)).Should(Succeed())

				// Wait for the spec changes to be reverted
				Eventually(func() *ocsv1.StorageClusterSpec {
					sc := scTemplate.DeepCopy()
					Expect(k8sClient.Get(ctx, scKey, sc)).Should(Succeed())
					return &sc.Spec
				}, timeout, interval).Should(Equal(spec))
			})
		})
		When("the storagecluster resource is modified while the reconcile strategy is set to strict", func() {
			It("should revert the changes and bring the resource back to its managed state", func() {
				// Set managed OCS to reconcile strategy to strict
				managedOCS := managedOCSTemplate.DeepCopy()
				Expect(k8sClient.Get(ctx, utils.GetResourceKey(managedOCS), managedOCS)).Should(Succeed())
				managedOCS.Spec.ReconcileStrategy = v1.ReconcileStrategyStrict
				Expect(k8sClient.Update(ctx, managedOCS)).Should(Succeed())

				// Get an updated storagecluster
				sc := scTemplate.DeepCopy()
				scKey := utils.GetResourceKey(sc)
				Expect(k8sClient.Get(ctx, scKey, sc)).Should(Succeed())

				// Update to empty spec
				spec := sc.Spec.DeepCopy()
				sc.Spec = ocsv1.StorageClusterSpec{}
				Expect(k8sClient.Update(ctx, sc)).Should(Succeed())

				// Wait for the spec changes to be reverted
				Eventually(func() *ocsv1.StorageClusterSpec {
					sc := scTemplate.DeepCopy()
					Expect(k8sClient.Get(ctx, scKey, sc)).Should(Succeed())
					return &sc.Spec
				}, timeout, interval).Should(Equal(spec))
			})
		})
		When("the storagecluster resource is modified while the reconcile strategy is set to none", func() {
			It("should not revert any changes back to the managed state", func() {
				// Set managed OCS to reconcile strategy to none
				managedOCS := managedOCSTemplate.DeepCopy()
				Expect(k8sClient.Get(ctx, utils.GetResourceKey(managedOCS), managedOCS)).Should(Succeed())
				managedOCS.Spec.ReconcileStrategy = v1.ReconcileStrategyNone
				Expect(k8sClient.Update(ctx, managedOCS)).Should(Succeed())

				// Get an updated storagecluster
				sc := scTemplate.DeepCopy()
				scKey := utils.GetResourceKey(sc)
				Expect(k8sClient.Get(ctx, scKey, sc)).Should(Succeed())

				// Update to empty spec
				sc.Spec = ocsv1.StorageClusterSpec{}
				Expect(k8sClient.Update(ctx, sc)).Should(Succeed())

				// Verify that the spec changes are not reverted
				Consistently(func() *ocsv1.StorageClusterSpec {
					sc := scTemplate.DeepCopy()
					Expect(k8sClient.Get(ctx, scKey, sc)).Should(Succeed())
					return &sc.Spec
				}, timeout, interval).Should(Equal(&sc.Spec))
			})
		})
		When("the prometheus resource is modified", func() {
			It("should revert the changes and bring the resource back to its managed state", func() {
				// Get an updated prometheus
				prom := promTemplate.DeepCopy()
				promKey := utils.GetResourceKey(prom)
				Expect(k8sClient.Get(ctx, promKey, prom)).Should(Succeed())

				// Update to empty spec
				spec := prom.Spec.DeepCopy()
				prom.Spec = promv1.PrometheusSpec{}
				Expect(k8sClient.Update(ctx, prom)).Should(Succeed())

				// Wait for the spec changes to be reverted
				Eventually(func() *promv1.PrometheusSpec {
					prom := promTemplate.DeepCopy()
					Expect(k8sClient.Get(ctx, promKey, prom)).Should(Succeed())
					return &prom.Spec
				}, timeout, interval).Should(Equal(spec))
			})
		})
		When("the prometheus resource is deleted", func() {
			It("should create a new prometheus in the namespace", func() {
				// Delete the prometheus resource
				Expect(k8sClient.Delete(ctx, promTemplate.DeepCopy())).Should(Succeed())

				// Wait for the prometheus to be recreated
				utils.WaitForResource(k8sClient, ctx, promTemplate.DeepCopy(), timeout, interval)
			})
		})
		When("the alertmanager resource is modified", func() {
			It("should revert the changes and bring the resource back to its managed state", func() {
				// Get an updated alertmanager
				am := amTemplate.DeepCopy()
				amKey := utils.GetResourceKey(am)
				Expect(k8sClient.Get(ctx, amKey, am)).Should(Succeed())

				// Update to empty spec
				spec := am.Spec.DeepCopy()
				am.Spec = promv1.AlertmanagerSpec{}
				Expect(k8sClient.Update(ctx, am)).Should(Succeed())

				// Wait for the spec changes to be reverted
				Eventually(func() *promv1.AlertmanagerSpec {
					am := amTemplate.DeepCopy()
					Expect(k8sClient.Get(ctx, amKey, am)).Should(Succeed())
					return &am.Spec
				}, timeout, interval).Should(Equal(spec))
			})
		})
		When("the alertmanager resource is deleted", func() {
			It("should create a new alertmanager in the namespace", func() {
				// Delete the alertmanager resource
				Expect(k8sClient.Delete(ctx, amTemplate.DeepCopy())).Should(Succeed())

				// Wait for the alertmanager to be recreated
				utils.WaitForResource(k8sClient, ctx, promTemplate.DeepCopy(), timeout, interval)
			})
		})
		When("there is no pagerduty secret in the cluster", func() {
			It("should not create alertmanager config", func() {
				setupAlertmanagerConfigConditions(false, false, true, true, true, true)

				// Ensure, over a period of time, that the resources are not created
				utils.EnsureNoResource(k8sClient, ctx, amConfigTemplate.DeepCopy(), timeout, interval)
			})
		})
		When("there is no value for PAGERDUTY_KEY in the pagerduty secret", func() {
			It("should not create alertmanager config", func() {
				setupAlertmanagerConfigConditions(true, false, true, true, true, true)

				// Ensure, over a period of time, that the resources are not created
				utils.EnsureNoResource(k8sClient, ctx, amConfigTemplate.DeepCopy(), timeout, interval)
			})
		})
		When("there is no deadmanssnitch secret in the cluster", func() {
			It("should not create alertmanager config", func() {
				setupAlertmanagerConfigConditions(true, true, false, false, true, true)

				// Ensure, over a period of time, that the resources are not created
				utils.EnsureNoResource(k8sClient, ctx, amConfigTemplate.DeepCopy(), timeout, interval)
			})
		})
		When("there is no value for SNITCH_URL in the deadmanssnitch secret", func() {
			It("should not create alertmanager config", func() {
				setupAlertmanagerConfigConditions(true, true, true, false, true, true)

				// Ensure, over a period of time, that the resources are not created
				utils.EnsureNoResource(k8sClient, ctx, amConfigTemplate.DeepCopy(), timeout, interval)
			})
		})
		When("there is no smtp secret in the cluster", func() {
			It("should not create alertmanager config", func() {
				setupAlertmanagerConfigConditions(true, true, true, true, false, false)

				// Ensure, over a period of time, that the resources are not created
				utils.EnsureNoResource(k8sClient, ctx, amConfigTemplate.DeepCopy(), timeout, interval)

			})
		})
		When("there is no value for the keys in smtp secret", func() {
			It("should not create alertmanager config", func() {
				setupAlertmanagerConfigConditions(true, true, true, true, true, false)

				// Ensure, over a period of time, that the resources are not created
				utils.EnsureNoResource(k8sClient, ctx, amConfigTemplate.DeepCopy(), timeout, interval)

			})
		})
		When("All conditions for creating an alertmanager config are met", func() {
			It("should create alertmanager config", func() {
				setupAlertmanagerConfigConditions(true, true, true, true, true, true)

				utils.WaitForResource(k8sClient, ctx, amConfigTemplate.DeepCopy(), timeout, interval)
			})
		})
		When("k8sMetricsServiceMonitor is modified", func() {
			It("should revert the changes and bring the resource back to its managed state", func() {
				// Wait for the resource to be created
				utils.WaitForResource(k8sClient, ctx, k8sMetricsServiceMonitorTemplate.DeepCopy(), timeout, interval)

				sm := k8sMetricsServiceMonitorTemplate.DeepCopy()
				smKey := utils.GetResourceKey(sm)
				Expect(k8sClient.Get(ctx, smKey, sm)).Should(Succeed())
				// Update the spec
				spec := sm.Spec.DeepCopy()
				sm.Spec = promv1.ServiceMonitorSpec{
					Endpoints: []promv1.Endpoint{},
				}
				Expect(k8sClient.Update(ctx, sm)).Should(Succeed())
				// Wait for the spec changes to be reverted
				Eventually(func() *promv1.ServiceMonitorSpec {
					sm := k8sMetricsServiceMonitorTemplate.DeepCopy()
					Expect(k8sClient.Get(ctx, smKey, sm)).Should(Succeed())
					return &sm.Spec
				}, timeout, interval).Should(Equal(spec))
			})
		})
		When("the dms prometheus rule resource is deleted", func() {
			It("should create a new dms prometheus rule in the namespace with expected labels", func() {
				// Ensure prometheus rule existed to begin with
				utils.WaitForResource(k8sClient, ctx, dmsPromRuleTemplate.DeepCopy(), timeout, interval)

				// Delete the prometheus rule resource
				Expect(k8sClient.Delete(ctx, dmsPromRuleTemplate.DeepCopy())).Should(Succeed())

				// Wait for the prometheus rule to be recreated
				utils.WaitForResource(k8sClient, ctx, dmsPromRuleTemplate.DeepCopy(), timeout, interval)

				// Check labels in prometheus rule
				Expect(k8sClient.Get(ctx, utils.GetResourceKey(&dmsPromRuleTemplate), &dmsPromRuleTemplate)).Should(Succeed())

				Expect(dmsPromRuleTemplate.Spec.Groups[0].Rules[0].Labels["alertname"]).Should(Equal("DeadMansSnitch"))
				Expect(dmsPromRuleTemplate.Spec.Groups[0].Rules[0].Labels["namespace"]).Should(Equal(testPrimaryNamespace))

			})
		})
		When("there is a pod monitor without an ocs-dedicated label", func() {
			It("should add the label to the pod monitor resource", func() {
				pm := podMonitorTemplate.DeepCopy()
				Expect(k8sClient.Create(ctx, pm)).Should(Succeed())

				Eventually(func() bool {
					return utils.ResourceHasLabel(k8sClient, ctx, pm, monLabelKey, monLabelValue)
				}, timeout, interval).Should(BeTrue())
			})
		})
		When("there is a service monitor without an ocs-dedicated label", func() {
			It("should add the label to the service monitor resource", func() {
				sm := serviceMonitorTemplate.DeepCopy()
				Expect(k8sClient.Create(ctx, sm)).Should(Succeed())

				Eventually(func() bool {
					return utils.ResourceHasLabel(k8sClient, ctx, sm, monLabelKey, monLabelValue)
				}, timeout, interval).Should(BeTrue())
			})
		})
		When("there is a prometheus rule without a ocs-dedicated label", func() {
			It("should add the label to the prometheus rule resource", func() {
				pr := promRuleTemplate.DeepCopy()
				Expect(k8sClient.Create(ctx, pr)).Should(Succeed())

				Eventually(func() bool {
					return utils.ResourceHasLabel(k8sClient, ctx, pr, monLabelKey, monLabelValue)
				}, timeout, interval).Should(BeTrue())
			})
		})
		When("the ocsInitialization resource is created", func() {
			It("should patch the ocsInitialization to enable ceph toolbox", func() {
				ocsInit := ocsInitializationTemplate.DeepCopy()
				Expect(k8sClient.Create(ctx, ocsInit)).Should(Succeed())

				ocsInitKey := utils.GetResourceKey(ocsInit)

				// Wait for the spec changes to be reverted
				Eventually(func() bool {
					Expect(k8sClient.Get(ctx, ocsInitKey, ocsInit)).Should(Succeed())
					return ocsInit.Spec.EnableCephTools
				}, timeout, interval).Should(Equal(true))
			})
		})
		When("the ocsInitialization resource is modified", func() {
			It("should revert the changes in ocsInitialization to enable ceph toolbox", func() {
				ocsInit := ocsInitializationTemplate.DeepCopy()
				ocsInitKey := utils.GetResourceKey(ocsInit)
				Expect(k8sClient.Get(ctx, ocsInitKey, ocsInit)).Should(Succeed())

				ocsInit.Spec.EnableCephTools = false
				Expect(k8sClient.Update(ctx, ocsInit)).Should(Succeed())

				// Wait for the spec changes to be reverted
				Eventually(func() bool {
					Expect(k8sClient.Get(ctx, ocsInitKey, ocsInit)).Should(Succeed())
					return ocsInit.Spec.EnableCephTools
				}, timeout, interval).Should(Equal(true))
			})
		})
		When("the OCS CSV resource is created", func() {
			It("should patch the OCS CSV to set resources for required pods", func() {
				ocsCSV := ocsCSVTemplate.DeepCopy()
				ocsCSVInitKey := utils.GetResourceKey(ocsCSV)
				Expect(k8sClient.Get(ctx, ocsCSVInitKey, ocsCSV)).Should(Succeed())

				deployments := ocsCSV.Spec.InstallStrategy.StrategySpec.DeploymentSpecs
				for i := range deployments {
					containers := deployments[i].Spec.Template.Spec.Containers
					for j := range containers {
						container := &containers[j]
						if container.Name == "ocs-operator" ||
							container.Name == "rook-ceph-operator" ||
							container.Name == "ocs-metrics-exporter" {
							Expect(container.Resources).Should(Equal(ctrlutils.GetResourceRequirements(container.Name)))
						}
					}
				}
			})
		})
		When("the OCS CSV resource is modified", func() {
			It("should revert the changes in OCS CSV to have provided resource requirements", func() {
				ocsCSV := ocsCSVTemplate.DeepCopy()
				ocsCSVInitKey := utils.GetResourceKey(ocsCSV)
				Expect(k8sClient.Get(ctx, ocsCSVInitKey, ocsCSV)).Should(Succeed())

				deployments := ocsCSV.Spec.InstallStrategy.StrategySpec.DeploymentSpecs
				var depIndex int
				var conIndex int
				for i := range deployments {
					containers := deployments[i].Spec.Template.Spec.Containers
					for j := range containers {
						container := &containers[j]
						if container.Name == "ocs-operator" {
							container.Resources.Limits = corev1.ResourceList{
								"cpu":    resource.MustParse("3000m"),
								"memory": resource.MustParse("8Gi"),
							}
							Expect(k8sClient.Update(ctx, ocsCSV)).Should(Succeed())
							depIndex = i
							conIndex = j
						}
					}
				}

				// Wait for the spec changes to be reverted
				Eventually(func() corev1.ResourceRequirements {
					Expect(k8sClient.Get(ctx, ocsCSVInitKey, ocsCSV)).Should(Succeed())
					deployment := ocsCSV.Spec.InstallStrategy.StrategySpec.DeploymentSpecs[depIndex]
					return deployment.Spec.Template.Spec.Containers[conIndex].Resources
				}, timeout, interval).Should(Equal(ctrlutils.GetResourceRequirements("ocs-operator")))
			})
		})
		When("replicas for noobaa-operator deployment are checked", func() {
			It("should have zero replicas", func() {
				mcgCSV := mcgCSVTemplate.DeepCopy()
				mcgCSVKey := utils.GetResourceKey(mcgCSV)
				Expect(k8sClient.Get(ctx, mcgCSVKey, mcgCSV)).Should(Succeed())
				deployments := mcgCSV.Spec.InstallStrategy.StrategySpec.DeploymentSpecs
				var index int
				for i := range deployments {
					deployment := deployments[i]
					if deployment.Name == "noobaa-deployment" {
						index = i
					}
				}
				desiredReplicas := int32(0)
				Eventually(func() int32 {
					Expect(k8sClient.Get(ctx, mcgCSVKey, mcgCSV)).Should(Succeed())
					deployment := mcgCSV.Spec.InstallStrategy.StrategySpec.DeploymentSpecs[index]
					return *deployment.Spec.Replicas
				}, timeout, interval).Should(Equal(desiredReplicas))

			})
		})
		When("there is a notification email address in the add-on parameter secret and smtp secret", func() {
			It("should update alertmanager email config with smtp details", func() {
				secret := addonParamsSecretTemplate.DeepCopy()
				secretKey := utils.GetResourceKey(secret)
				Expect(k8sClient.Get(ctx, secretKey, secret)).Should(Succeed())

				// Update notification email to the addon param secret
				secret.Data["notification-email-0"] = []byte("test-0@email.com")
				Expect(k8sClient.Update(ctx, secret)).Should(Succeed())

				// Wait for alertmanager to get updated with smtp details
				amconfig := amConfigTemplate.DeepCopy()
				amconfigKey := utils.GetResourceKey(amconfig)
				utils.WaitForAlertManagerSMTPReceiverEmailConfigToUpdate(
					k8sClient,
					ctx,
					amconfigKey,
					[]string{"test-0@email.com"},
					"SendGrid",
					timeout,
					interval,
				)
			})
		})
		When("notification email address in the add-on parameter is updated", func() {
			It("should update alertmanager config with the updated notification email", func() {
				secret := addonParamsSecretTemplate.DeepCopy()
				secretKey := utils.GetResourceKey(secret)
				Expect(k8sClient.Get(ctx, secretKey, secret)).Should(Succeed())

				// Update notification email in addon param secret
				secret.Data["notification-email-0"] = []byte("test-new@email.com")
				Expect(k8sClient.Update(ctx, secret)).Should(Succeed())

				// Wait for alertmanager to get updated with smtp details
				amconfig := amConfigTemplate.DeepCopy()
				amconfigKey := utils.GetResourceKey(amconfig)
				utils.WaitForAlertManagerSMTPReceiverEmailConfigToUpdate(
					k8sClient,
					ctx,
					amconfigKey,
					[]string{"test-new@email.com"},
					"SendGrid",
					timeout,
					interval,
				)
			})
		})
		When("second notification email addresses is added in add-on parameter", func() {
			It("should update alertmanager config with the added notification email id", func() {
				secret := addonParamsSecretTemplate.DeepCopy()
				secretKey := utils.GetResourceKey(secret)
				Expect(k8sClient.Get(ctx, secretKey, secret)).Should(Succeed())

				// Update notification email in addon param secret
				secret.Data["notification-email-0"] = []byte("test-0@email.com")
				secret.Data["notification-email-1"] = []byte("test-1@email.com")
				Expect(k8sClient.Update(ctx, secret)).Should(Succeed())

				// Wait for alertmanager to get updated with smtp details
				amconfig := amConfigTemplate.DeepCopy()
				amconfigKey := utils.GetResourceKey(amconfig)
				utils.WaitForAlertManagerSMTPReceiverEmailConfigToUpdate(
					k8sClient,
					ctx,
					amconfigKey,
					[]string{"test-0@email.com", "test-1@email.com"},
					"SendGrid",
					timeout,
					interval,
				)
			})
		})
		When("second notification email address in the add-on parameter is removed", func() {
			It("should update alertmanager config by removing the second email address", func() {
				secret := addonParamsSecretTemplate.DeepCopy()
				secretKey := utils.GetResourceKey(secret)
				Expect(k8sClient.Get(ctx, secretKey, secret)).Should(Succeed())

				// remove notification email from addon param secret
				secret.Data["notification-email-1"] = []byte("")
				Expect(k8sClient.Update(ctx, secret)).Should(Succeed())

				// Wait for alertmanager to get updated with smtp details
				amconfig := amConfigTemplate.DeepCopy()
				amconfigKey := utils.GetResourceKey(amconfig)
				utils.WaitForAlertManagerSMTPReceiverEmailConfigToUpdate(
					k8sClient,
					ctx,
					amconfigKey,
					[]string{"test-0@email.com"},
					"SendGrid",
					timeout,
					interval,
				)
			})
		})
		When("there is no notification email address in the add-on parameter", func() {
			It("should update alertmanager config by removing the SMTP email configs", func() {
				secret := addonParamsSecretTemplate.DeepCopy()
				secretKey := utils.GetResourceKey(secret)
				Expect(k8sClient.Get(ctx, secretKey, secret)).Should(Succeed())

				// remove notification email from addon param secret
				secret.Data["notification-email-0"] = []byte("")
				Expect(k8sClient.Update(ctx, secret)).Should(Succeed())

				// Wait for alertmanager to remove the email configs
				amconfig := amConfigTemplate.DeepCopy()
				amconfigKey := utils.GetResourceKey(amconfig)
				utils.WaitForAlertManagerSMTPReceiverEmailConfigToUpdate(
					k8sClient,
					ctx,
					amconfigKey,
					[]string{},
					"SendGrid",
					timeout,
					interval,
				)
			})
		})
		When("the EgressNetworkPolicy resource is modified", func() {
			It("should revert the changes and bring the resource back to its managed state", func() {
				if !testReconciler.AvailableCRDs[EgressNetworkPolicyCRD] {
					Skip("Skipping this test as EgressNetworkPolicy is not present")
				}
				// Get an updated EgressNetworkPolicy
				egress := egressNetworkPolicyTemplate.DeepCopy()
				egressKey := utils.GetResourceKey(egress)
				Expect(k8sClient.Get(ctx, egressKey, egress)).Should(Succeed())

				// Update to empty spec
				spec := egress.Spec.DeepCopy()
				egress.Spec = openshiftv1.EgressNetworkPolicySpec{
					Egress: []openshiftv1.EgressNetworkPolicyRule{},
				}
				Expect(k8sClient.Update(ctx, egress)).Should(Succeed())

				// Wait for the spec changes to be reverted
				Eventually(func() *openshiftv1.EgressNetworkPolicySpec {
					egress := egressNetworkPolicyTemplate.DeepCopy()
					Expect(k8sClient.Get(ctx, egressKey, egress)).Should(Succeed())
					return &egress.Spec
				}, timeout, interval).Should(Equal(spec))
			})
		})
		When("the SNITCH_URL value in dms secret is modified", func() {
			It("should update the EgressNetworkPolicy resource with the new snitch domain", func() {
				if !testReconciler.AvailableCRDs[EgressNetworkPolicyCRD] {
					Skip("Skipping this test as EgressNetworkPolicy is not present")
				}
				dmsSecret := dmsSecretTemplate.DeepCopy()
				Expect(k8sClient.Get(ctx, utils.GetResourceKey(dmsSecret), dmsSecret)).Should(Succeed())
				dmsSecret.Data["SNITCH_URL"] = []byte("https://test.in/4a029adb4c")
				Expect(k8sClient.Update(ctx, dmsSecret)).Should(Succeed())

				Eventually(func() bool {
					egress := egressNetworkPolicyTemplate.DeepCopy()
					Expect(k8sClient.Get(ctx, utils.GetResourceKey(egress), egress)).Should(Succeed())
					egressRules := egress.Spec.Egress
					for i := range egressRules {
						egressRule := &egressRules[i]
						if egressRule.To.DNSName == "test.in" {
							return true
						}
					}
					return false
				}, timeout, interval).Should(Equal(true))
			})
		})
		When("the host value in smtp secret is modified", func() {
			It("should update the EgressNetworkPolicy resource with the new host", func() {
				if !testReconciler.AvailableCRDs[EgressNetworkPolicyCRD] {
					Skip("Skipping this test as EgressNetworkPolicy is not present")
				}
				smtpSecret := smtpSecretTemplate.DeepCopy()
				Expect(k8sClient.Get(ctx, utils.GetResourceKey(smtpSecret), smtpSecret)).Should(Succeed())
				smtpSecret.Data["host"] = []byte("test-host-2")
				Expect(k8sClient.Update(ctx, smtpSecret)).Should(Succeed())

				Eventually(func() bool {
					egress := egressNetworkPolicyTemplate.DeepCopy()
					Expect(k8sClient.Get(ctx, utils.GetResourceKey(egress), egress)).Should(Succeed())
					egressRules := egress.Spec.Egress
					for i := range egressRules {
						egressRule := &egressRules[i]
						if egressRule.To.DNSName == "test-host-2" {
							return true
						}
					}
					return false
				}, timeout, interval).Should(Equal(true))
			})
		})
		When("the storageProviderEndpoint in addon params is modified", func() {
			It("should update the EgressNetworkPolicy resource with the new host", func() {
				if !testReconciler.AvailableCRDs[EgressNetworkPolicyCRD] {
					Skip("Skipping this test as EgressNetworkPolicy is not present")
				}
				if testReconciler.DeploymentType != consumerDeploymentType {
					Skip(fmt.Sprintf("Skipping this test for %v deployment", testReconciler.DeploymentType))
				}
				addonParamsSecret := addonParamsSecretTemplate.DeepCopy()
				Expect(k8sClient.Get(ctx, utils.GetResourceKey(addonParamsSecret), addonParamsSecret)).Should(Succeed())
				addonParamsSecret.Data[storageProviderEndpointKey] = []byte("myLoadBalancerHostName:50051")
				Expect(k8sClient.Update(ctx, addonParamsSecret)).Should(Succeed())

				Eventually(func() bool {
					egress := egressNetworkPolicyTemplate.DeepCopy()
					Expect(k8sClient.Get(ctx, utils.GetResourceKey(egress), egress)).Should(Succeed())
					egressRules := egress.Spec.Egress
					for i := range egressRules {
						egressRule := &egressRules[i]
						if egressRule.To.DNSName == "myLoadBalancerHostName" {
							return true
						}
					}
					return false
				}, timeout, interval).Should(Equal(true))
			})
		})
		When("the EgressNetworkPolicy resource is deleted", func() {
			It("should create a new EgressNetworkPolicy in the namespace", func() {
				if !testReconciler.AvailableCRDs[EgressNetworkPolicyCRD] {
					Skip("Skipping this test as EgressNetworkPolicy is not present")
				}
				// Delete the EgressNetworkPolicy resource
				Expect(k8sClient.Delete(ctx, egressNetworkPolicyTemplate.DeepCopy())).Should(Succeed())

				// Wait for the EgressNetworkPolicy to be recreated
				utils.WaitForResource(k8sClient, ctx, egressNetworkPolicyTemplate.DeepCopy(), timeout, interval)
			})
		})

		When("the EgressFirewall resource is modified", func() {
			It("should revert the changes and bring the resource back to its managed state", func() {
				if !testReconciler.AvailableCRDs[EgressFirewallCRD] {
					Skip("Skipping this test as EgressFirewall is not present")
				}
				// Get an updated EgressFirewall
				egress := egressFirewallTemplate.DeepCopy()
				egressKey := utils.GetResourceKey(egress)
				Expect(k8sClient.Get(ctx, egressKey, egress)).Should(Succeed())

				// Update to empty spec
				spec := egress.Spec.DeepCopy()
				egress.Spec = ovnv1.EgressFirewallSpec{
					Egress: []ovnv1.EgressFirewallRule{},
				}
				Expect(k8sClient.Update(ctx, egress)).Should(Succeed())

				// Wait for the spec changes to be reverted
				Eventually(func() *ovnv1.EgressFirewallSpec {
					egress := egressFirewallTemplate.DeepCopy()
					Expect(k8sClient.Get(ctx, egressKey, egress)).Should(Succeed())
					return &egress.Spec
				}, timeout, interval).Should(Equal(spec))
			})
		})
		When("the SNITCH_URL value in dms secret is modified", func() {
			It("should update the EgressFirewall resource with the new snitch domain", func() {
				if !testReconciler.AvailableCRDs[EgressFirewallCRD] {
					Skip("Skipping this test as EgressFirewall is not present")
				}
				dmsSecret := dmsSecretTemplate.DeepCopy()
				Expect(k8sClient.Get(ctx, utils.GetResourceKey(dmsSecret), dmsSecret)).Should(Succeed())
				dmsSecret.Data["SNITCH_URL"] = []byte("https://test.in/4a029adb4c")
				Expect(k8sClient.Update(ctx, dmsSecret)).Should(Succeed())

				Eventually(func() bool {
					egress := egressFirewallTemplate.DeepCopy()
					Expect(k8sClient.Get(ctx, utils.GetResourceKey(egress), egress)).Should(Succeed())
					egressRules := egress.Spec.Egress
					for i := range egressRules {
						egressRule := &egressRules[i]
						if egressRule.To.DNSName == "test.in" {
							return true
						}
					}
					return false
				}, timeout, interval).Should(Equal(true))
			})
		})
		When("the host value in smtp secret is modified", func() {
			It("should update the EgressFirewall resource with the new host", func() {
				if !testReconciler.AvailableCRDs[EgressFirewallCRD] {
					Skip("Skipping this test as EgressFirewall is not present")
				}
				smtpSecret := smtpSecretTemplate.DeepCopy()
				Expect(k8sClient.Get(ctx, utils.GetResourceKey(smtpSecret), smtpSecret)).Should(Succeed())
				smtpSecret.Data["host"] = []byte("test-host-2")
				Expect(k8sClient.Update(ctx, smtpSecret)).Should(Succeed())

				Eventually(func() bool {
					egress := egressFirewallTemplate.DeepCopy()
					Expect(k8sClient.Get(ctx, utils.GetResourceKey(egress), egress)).Should(Succeed())
					egressRules := egress.Spec.Egress
					for i := range egressRules {
						egressRule := &egressRules[i]
						if egressRule.To.DNSName == "test-host-2" {
							return true
						}
					}
					return false
				}, timeout, interval).Should(Equal(true))
			})
		})
		When("the storageProviderEndpoint in addon params is modified", func() {
			It("should update the EgressFirewall resource with the new host", func() {
				if !testReconciler.AvailableCRDs[EgressFirewallCRD] {
					Skip("Skipping this test as EgressFirewall is not present")
				}
				if testReconciler.DeploymentType != consumerDeploymentType {
					Skip(fmt.Sprintf("Skipping this test for %v deployment", testReconciler.DeploymentType))
				}
				addonParamsSecret := addonParamsSecretTemplate.DeepCopy()
				Expect(k8sClient.Get(ctx, utils.GetResourceKey(addonParamsSecret), addonParamsSecret)).Should(Succeed())
				addonParamsSecret.Data[storageProviderEndpointKey] = []byte("myLoadBalancerHostName:50051")
				Expect(k8sClient.Update(ctx, addonParamsSecret)).Should(Succeed())

				Eventually(func() bool {
					egress := egressFirewallTemplate.DeepCopy()
					Expect(k8sClient.Get(ctx, utils.GetResourceKey(egress), egress)).Should(Succeed())
					egressRules := egress.Spec.Egress
					for i := range egressRules {
						egressRule := &egressRules[i]
						if egressRule.To.DNSName == "myLoadBalancerHostName" {
							return true
						}
					}
					return false
				}, timeout, interval).Should(Equal(true))
			})
		})
		When("the EgressFirewall resource is deleted", func() {
			It("should create a new EgressFirewall in the namespace", func() {
				if !testReconciler.AvailableCRDs[EgressFirewallCRD] {
					Skip("Skipping this test as EgressFirewall is not present")
				}
				// Delete the EgressFirewall resource
				Expect(k8sClient.Delete(ctx, egressFirewallTemplate.DeepCopy())).Should(Succeed())

				// Wait for the EgressFirewall to be recreated
				utils.WaitForResource(k8sClient, ctx, egressFirewallTemplate.DeepCopy(), timeout, interval)
			})
		})

		When("the ingress NetworkPolicy resource is modified", func() {
			It("should revert the changes and bring the resource back to its managed state", func() {
				// Get an updated NetworkPolicy
				ingress := ingressNetworkPolicyTemplate.DeepCopy()
				ingressKey := utils.GetResourceKey(ingress)
				Expect(k8sClient.Get(ctx, ingressKey, ingress)).Should(Succeed())

				// Update to empty spec
				spec := ingress.Spec.DeepCopy()
				ingress.Spec = netv1.NetworkPolicySpec{}
				Expect(k8sClient.Update(ctx, ingress)).Should(Succeed())

				// Wait for the spec changes to be reverted
				Eventually(func() *netv1.NetworkPolicySpec {
					ingress := ingressNetworkPolicyTemplate.DeepCopy()
					Expect(k8sClient.Get(ctx, ingressKey, ingress)).Should(Succeed())
					return &ingress.Spec
				}, timeout, interval).Should(Equal(spec))
			})
		})
		When("the ingress NetworkPolicy resource is deleted", func() {
			It("should create a new ingress NetworkPolicy in the namespace", func() {
				// Delete the NetworkPolicy resource
				Expect(k8sClient.Delete(ctx, ingressNetworkPolicyTemplate.DeepCopy())).Should(Succeed())

				// Wait for the NetworkPolicy to be recreated
				utils.WaitForResource(k8sClient, ctx, ingressNetworkPolicyTemplate.DeepCopy(), timeout, interval)
			})
		})
		When("the ceph ingress NetworkPolicy resource is modified", func() {
			It("should revert the changes and bring the resource back to its managed state", func() {
				// Get an updated NetworkPolicy
				cephIngress := cephIngressNetworkPolicyTemplate.DeepCopy()
				cephIngressKey := utils.GetResourceKey(cephIngress)
				Expect(k8sClient.Get(ctx, cephIngressKey, cephIngress)).Should(Succeed())

				// Update to empty spec
				spec := cephIngress.Spec.DeepCopy()
				cephIngress.Spec = netv1.NetworkPolicySpec{}
				Expect(k8sClient.Update(ctx, cephIngress)).Should(Succeed())

				// Wait for the spec changes to be reverted
				Eventually(func() *netv1.NetworkPolicySpec {
					cephIngress := cephIngressNetworkPolicyTemplate.DeepCopy()
					Expect(k8sClient.Get(ctx, cephIngressKey, cephIngress)).Should(Succeed())
					return &cephIngress.Spec
				}, timeout, interval).Should(Equal(spec))
			})
		})
		When("the ceph ingress NetworkPolicy resource is deleted", func() {
			It("should create a new ingress NetworkPolicy in the namespace", func() {
				// Delete the NetworkPolicy resource
				Expect(k8sClient.Delete(ctx, cephIngressNetworkPolicyTemplate.DeepCopy())).Should(Succeed())

				// Wait for the NetworkPolicy to be recreated
				utils.WaitForResource(k8sClient, ctx, cephIngressNetworkPolicyTemplate.DeepCopy(), timeout, interval)
			})
		})
		When("the provider api server ingress NetworkPolicy resource is modified", func() {
			It("should create a new ingress NetworkPolicy in the namespace", func() {
				if testReconciler.DeploymentType != providerDeploymentType {
					Skip(fmt.Sprintf("Skipping this test for %v deployment", testReconciler.DeploymentType))
				}

				// Delete the NetworkPolicy resource
				Expect(k8sClient.Delete(ctx, providerApiServerIngressNetworkPolicyTemplate.DeepCopy())).Should(Succeed())

				// Wait for the NetworkPolicy to be recreated
				utils.WaitForResource(k8sClient, ctx, providerApiServerIngressNetworkPolicyTemplate.DeepCopy(), timeout, interval)
			})
		})
		When("the provider api server ingress  NetworkPolicy resource is deleted", func() {
			It("should create a new ingress NetworkPolicy in the namespace", func() {
				if testReconciler.DeploymentType != providerDeploymentType {
					Skip(fmt.Sprintf("Skipping this test for %v deployment", testReconciler.DeploymentType))
				}
				// Delete the NetworkPolicy resource
				Expect(k8sClient.Delete(ctx, providerApiServerIngressNetworkPolicyTemplate.DeepCopy())).Should(Succeed())

				// Wait for the NetworkPolicy to be recreated
				utils.WaitForResource(k8sClient, ctx, providerApiServerIngressNetworkPolicyTemplate.DeepCopy(), timeout, interval)
			})
		})
		When("the prometheus proxy ingress NetworkPolicy resource is modified", func() {
			It("should revert the changes and bring the resource back to its managed state", func() {
				// Get an updated NetworkPolicy
				prometheusProxyIngress := prometheusProxyIngressNetworkPolicyTemplate.DeepCopy()
				prometheusProxyIngressKey := utils.GetResourceKey(prometheusProxyIngress)
				Expect(k8sClient.Get(ctx, prometheusProxyIngressKey, prometheusProxyIngress)).Should(Succeed())

				// Update to empty spec
				spec := prometheusProxyIngress.Spec.DeepCopy()
				prometheusProxyIngress.Spec = netv1.NetworkPolicySpec{}
				Expect(k8sClient.Update(ctx, prometheusProxyIngress)).Should(Succeed())

				// Wait for the spec changes to be reverted
				Eventually(func() *netv1.NetworkPolicySpec {
					prometheusProxyIngress := prometheusProxyIngressNetworkPolicyTemplate.DeepCopy()
					Expect(k8sClient.Get(ctx, prometheusProxyIngressKey, prometheusProxyIngress)).Should(Succeed())
					return &prometheusProxyIngress.Spec
				}, timeout, interval).Should(Equal(spec))
			})
		})
		When("the prometheus proxy ingress NetworkPolicy resource is deleted", func() {
			It("should create a new ingress NetworkPolicy in the namespace", func() {
				// Delete the NetworkPolicy resource
				Expect(k8sClient.Delete(ctx, prometheusProxyIngressNetworkPolicyTemplate.DeepCopy())).Should(Succeed())

				// Wait for the NetworkPolicy to be recreated
				utils.WaitForResource(k8sClient, ctx, prometheusProxyIngressNetworkPolicyTemplate.DeepCopy(), timeout, interval)
			})
		})
		When("the addon config map does not exist while all other uninstall conditions are met", func() {
			It("should not delete the managedOCS resource", func() {
				setupUninstallConditions(false, testAddonConfigMapDeleteLabelKey, true, true, true, false, false, false)

				managedOCS := managedOCSTemplate.DeepCopy()
				key := utils.GetResourceKey((managedOCS))
				Consistently(func() error {
					return k8sClient.Get(ctx, key, managedOCS)
				}, timeout, interval).Should(Succeed())
			})
		})
		When("the addon config map does not have a delete label while all other uninstall conditions are met", func() {
			It("should not delete the managedOCS resource", func() {
				setupUninstallConditions(true, "", true, true, true, false, false, false)

				managedOCS := managedOCSTemplate.DeepCopy()
				key := utils.GetResourceKey((managedOCS))
				Consistently(func() error {
					return k8sClient.Get(ctx, key, managedOCS)
				}, timeout, interval).Should(Succeed())
			})
		})
		When("the addon config map does not have a valid delete label while all other uninstall conditions are met", func() {
			It("should not delete the managedOCS resource", func() {
				setupUninstallConditions(true, "invalid-label", true, true, true, false, false, false)

				managedOCS := managedOCSTemplate.DeepCopy()
				key := utils.GetResourceKey((managedOCS))
				Consistently(func() error {
					return k8sClient.Get(ctx, key, managedOCS)
				}, timeout, interval).Should(Succeed())
			})
		})
		When("the storagecluster is not ready while all other uninstall conditions are met", func() {
			It("should not delete the managedOCS resource", func() {
				setupUninstallConditions(true, testAddonConfigMapDeleteLabelKey, false, true, true, false, false, false)

				managedOCS := managedOCSTemplate.DeepCopy()
				key := utils.GetResourceKey((managedOCS))
				Consistently(func() error {
					return k8sClient.Get(ctx, key, managedOCS)
				}, timeout, interval).Should(Succeed())
			})
		})
		When("prometheus is not ready while all other uninstall conditions are met", func() {
			It("should not delete the managedOCS resource", func() {
				setupUninstallConditions(true, testAddonConfigMapDeleteLabelKey, true, false, true, false, false, false)

				managedOCS := managedOCSTemplate.DeepCopy()
				key := utils.GetResourceKey((managedOCS))
				Consistently(func() error {
					return k8sClient.Get(ctx, key, managedOCS)
				}, timeout, interval).Should(Succeed())
			})
		})
		When("alertmanager is not ready while all other uninstall conditions are met", func() {
			It("should not delete the managedOCS resource", func() {
				setupUninstallConditions(true, testAddonConfigMapDeleteLabelKey, true, true, false, false, false, false)

				managedOCS := managedOCSTemplate.DeepCopy()
				key := utils.GetResourceKey((managedOCS))
				Consistently(func() error {
					return k8sClient.Get(ctx, key, managedOCS)
				}, timeout, interval).Should(Succeed())
			})
		})
		When("there are PVs in the primary namespace while all other uninstall conditions are met", func() {
			It("should not delete the managedOCS resource", func() {
				setupUninstallConditions(true, testAddonConfigMapDeleteLabelKey, true, true, false, true, false, false)

				managedOCS := managedOCSTemplate.DeepCopy()
				key := utils.GetResourceKey((managedOCS))
				Consistently(func() error {
					return k8sClient.Get(ctx, key, managedOCS)
				}, timeout, interval).Should(Succeed())
			})
		})
		When("there are PVs in a secondary namespace while all other uninstall conditions are met", func() {
			It("should not delete the managedOCS resource", func() {
				setupUninstallConditions(true, testAddonConfigMapDeleteLabelKey, true, true, false, false, true, false)

				managedOCS := managedOCSTemplate.DeepCopy()
				key := utils.GetResourceKey((managedOCS))
				Consistently(func() error {
					return k8sClient.Get(ctx, key, managedOCS)
				}, timeout, interval).Should(Succeed())
			})
		})
		When("there are storage consumers in provider cluster while all other uninstall conditions are met", func() {
			It("should not delete the managedOCS resource", func() {
				if testReconciler.DeploymentType != providerDeploymentType {
					Skip(fmt.Sprintf("Skipping the test as it is not required by %v", testReconciler.DeploymentType))
				}
				setupUninstallConditions(true, testAddonConfigMapDeleteLabelKey, true, true, true, false, false, true)

				managedOCS := managedOCSTemplate.DeepCopy()
				key := utils.GetResourceKey((managedOCS))
				if testReconciler.DeploymentType == providerDeploymentType {
					Consistently(func() error {
						return k8sClient.Get(ctx, key, managedOCS)
					}, timeout, interval).Should(Succeed())
				}
			})
		})
		When("All uninstall conditions are met", func() {
			It("should delete the managedOCS", func() {
				setupUninstallConditions(true, testAddonConfigMapDeleteLabelKey, true, true, true, false, false, false)

				managedOCS := managedOCSTemplate.DeepCopy()
				key := utils.GetResourceKey((managedOCS))
				Eventually(func() bool {
					err := k8sClient.Get(ctx, key, managedOCS)
					return err != nil && errors.IsNotFound(err)
				}, timeout, interval).Should(BeTrue())
			})
			It("should delete the deployer csv", func() {
				csv := csvTemplate.DeepCopy()
				key := utils.GetResourceKey(csv)
				Eventually(func() bool {
					err := k8sClient.Get(ctx, key, csv)
					return err != nil && errors.IsNotFound(err)
				}, timeout, interval).Should(BeTrue())
			})
		})
	})
})

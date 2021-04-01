package controllers

import (
	"context"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	ocsv1 "github.com/openshift/ocs-operator/pkg/apis/ocs/v1"
	v1 "github.com/openshift/ocs-osd-deployer/api/v1alpha1"
	utils "github.com/openshift/ocs-osd-deployer/testutils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("ManagedOCS controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		ManagedOCSName = "test-managedocs"
		TestNamespace  = "default"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("reconcile()", func() {
		When("there is no add-on parameters secret in the cluster", func() {
			It("should not create a new storage cluster", func() {
				ctx := context.Background()
				scList := &ocsv1.StorageClusterList{}

				Expect(k8sClient.List(ctx, scList, client.InNamespace(TestNamespace))).Should(Succeed())
				Expect(scList.Items).Should(HaveLen(0))

				// addon param secret does not exist
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      TestAddOnParamsSecretName,
						Namespace: TestNamespace,
					},
				}
				Expect(k8sClient.Get(ctx, utils.GetResourceKey(secret), secret)).Should(
					WithTransform(errors.IsNotFound, BeTrue()))
				managedOCS := &v1.ManagedOCS{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ManagedOCSName,
						Namespace: TestNamespace,
					},
				}
				Expect(k8sClient.Create(ctx, managedOCS)).Should(Succeed())
				Expect(k8sClient.Get(ctx, utils.GetResourceKey(managedOCS), managedOCS)).Should(Succeed())

				sc := &ocsv1.StorageCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      storageClusterName,
						Namespace: TestNamespace,
					},
				}
				utils.EnsureNoResource(k8sClient, ctx, sc, timeout, interval)
			})
		})
		When("there is incorrect data in the add-on parameters secret", func() {
			It("should not create a new storage cluster", func() {
				ctx := context.Background()

				scList := &ocsv1.StorageClusterList{}
				Expect(k8sClient.List(ctx, scList, client.InNamespace(TestNamespace))).Should(Succeed())
				Expect(scList.Items).Should(HaveLen(0))

				// Create the secret
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      TestAddOnParamsSecretName,
						Namespace: TestNamespace,
					},
					Data: map[string][]byte{
						"size": []byte("AA"),
					},
				}
				Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
				Expect(k8sClient.Get(ctx, utils.GetResourceKey(secret), secret)).Should(Succeed())

				managedOCS := &v1.ManagedOCS{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ManagedOCSName,
						Namespace: TestNamespace,
					},
				}
				Expect(k8sClient.Get(ctx, utils.GetResourceKey(managedOCS), managedOCS)).Should(Succeed())

				// No storage cluster should be created
				sc := &ocsv1.StorageCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      storageClusterName,
						Namespace: TestNamespace,
					},
				}
				utils.EnsureNoResource(k8sClient, ctx, sc, timeout, interval)

				// Remove the secret for future cases
				Expect(k8sClient.Delete(context.Background(), secret)).Should(Succeed())
			})
		})

		When("there is no storage cluster resource in the cluster", func() {
			It("should create a new storage cluster", func() {
				ctx := context.Background()

				scList := &ocsv1.StorageClusterList{}
				Expect(k8sClient.List(ctx, scList, client.InNamespace(TestNamespace))).Should(Succeed())
				Expect(scList.Items).Should(HaveLen(0))

				// Create the secret
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      TestAddOnParamsSecretName,
						Namespace: TestNamespace,
					},
					Data: map[string][]byte{
						"size": []byte("1"),
					},
				}
				Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
				Expect(k8sClient.Get(ctx, utils.GetResourceKey(secret), secret)).Should(Succeed())

				managedOCS := &v1.ManagedOCS{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ManagedOCSName,
						Namespace: TestNamespace,
					},
				}
				Expect(k8sClient.Get(ctx, utils.GetResourceKey(managedOCS), managedOCS)).Should(Succeed())

				sc := &ocsv1.StorageCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      storageClusterName,
						Namespace: TestNamespace,
					},
				}
				utils.WaitForResource(k8sClient, ctx, sc, timeout, interval)
			})
		})

		When("the storeage cluster is not ready", func() {
			ctx := context.Background()
			managedOCS := &v1.ManagedOCS{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ManagedOCSName,
					Namespace: TestNamespace,
				},
			}

			BeforeEach(func() {
				// This test, like the ones below it, assume managed-ocs is already created.
				Expect(k8sClient.Get(ctx, utils.GetResourceKey(managedOCS), managedOCS)).Should(Succeed())

				// Ensure that the storage cluster is not ready
				// This test, like the ones below it, assume a StorageCluster is already created.
				sc := ocsv1.StorageCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      storageClusterName,
						Namespace: TestNamespace,
					},
				}
				Expect(k8sClient.Get(ctx, utils.GetResourceKey(&sc), &sc)).Should(Succeed())

				// Updating the Status of the StorageCluster should trigger a reconcile
				// for managed-ocs
				sc.Status.Phase = "Pending"
				Expect(k8sClient.Status().Update(ctx, &sc)).Should(Succeed())
			})

			It("should update its installation status", func() {
				By("reflecting the sc status in the managed-ocs cr")
				Eventually(func() v1.ComponentState {
					Expect(k8sClient.Get(ctx, utils.GetResourceKey(managedOCS), managedOCS)).Should(Succeed())
					return managedOCS.Status.Components.StorageCluster.State
				}, timeout, interval).Should(Equal(v1.ComponentPending))
			})
		})

		When("the storeage cluster is ready", func() {
			ctx := context.Background()
			managedOCS := &v1.ManagedOCS{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ManagedOCSName,
					Namespace: TestNamespace,
				},
			}

			BeforeEach(func() {
				// This test, like the ones below it, assume managed-ocs is already created.
				Expect(k8sClient.Get(ctx, utils.GetResourceKey(managedOCS), managedOCS)).Should(Succeed())

				// Ensure that the storage cluster is not ready
				// This test, like the ones below it, assume a StorageCluster is already created.
				sc := ocsv1.StorageCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      storageClusterName,
						Namespace: TestNamespace,
					},
				}
				Expect(k8sClient.Get(ctx, utils.GetResourceKey(&sc), &sc)).Should(Succeed())

				// Updating the Status of the StorageCluster should trigger a reconcile
				// for managed-ocs
				sc.Status.Phase = "Ready"
				Expect(k8sClient.Status().Update(ctx, &sc)).Should(Succeed())
			})

			It("should update its installation status", func() {
				By("reflecting the sc status in the managed-ocs cr")
				Eventually(func() v1.ComponentState {
					Expect(k8sClient.Get(ctx, utils.GetResourceKey(managedOCS), managedOCS)).Should(Succeed())
					return managedOCS.Status.Components.StorageCluster.State
				}, timeout, interval).Should(Equal(v1.ComponentReady))
			})
		})

		When("the storage cluster is deleted", func() {
			It("should create a new storage cluster in the namespace", func() {
				ctx := context.Background()
				sc := &ocsv1.StorageCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      storageClusterName,
						Namespace: TestNamespace,
					},
				}
				Expect(k8sClient.Get(ctx, utils.GetResourceKey(sc), sc)).Should(Succeed())

				// Delete the strorage cluster
				Expect(k8sClient.Delete(ctx, sc)).Should(Succeed())
				// Race condition: this needs to occur before reconciliation loop runs.
				Expect(k8sClient.Get(ctx, utils.GetResourceKey(sc), sc)).Should(
					WithTransform(errors.IsNotFound, BeTrue()))

				// Wait for the storage cluster to be re created
				utils.WaitForResource(k8sClient, ctx, sc, timeout, interval)
			})
		})

		When("the storage cluster is modified while in strict mode", func() {
			It("should revert the changes and bring the storage cluster back to its managed state", func() {
				ctx := context.Background()

				// Verify strict mode
				managedOCS := &v1.ManagedOCS{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ManagedOCSName,
						Namespace: TestNamespace,
					},
				}
				Expect(k8sClient.Get(ctx, utils.GetResourceKey(managedOCS), managedOCS)).Should(Succeed())
				Expect(managedOCS.Status.ReconcileStrategy == v1.ReconcileStrategyStrict).Should(BeTrue())

				sc := &ocsv1.StorageCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      storageClusterName,
						Namespace: TestNamespace,
					},
				}
				scKey := utils.GetResourceKey(sc)
				Expect(k8sClient.Get(ctx, scKey, sc)).Should(Succeed())

				// Modify the storagecluster spec
				spec := sc.Spec.DeepCopy()
				sc.Spec = ocsv1.StorageClusterSpec{}
				Expect(k8sClient.Update(ctx, sc)).Should(Succeed())

				// Wait for the storage cluster to be modfied again to reflect that it was reconciled
				utils.WaitForResourceSpecToUpdate(k8sClient, ctx, sc, timeout, interval)

				// Verify that the storage cluster was reverted to its original state
				Expect(reflect.DeepEqual(sc.Spec, *spec)).Should(BeTrue())
			})
		})

		When("the sds count is increased in secret", func() {
			It("should increase storage cluster size", func() {
				ctx := context.Background()

				managedOCS := &v1.ManagedOCS{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ManagedOCSName,
						Namespace: TestNamespace,
					},
				}
				managedOCSKey := utils.GetResourceKey(managedOCS)
				Expect(k8sClient.Get(ctx, managedOCSKey, managedOCS)).Should(Succeed())

				sc := &ocsv1.StorageCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      storageClusterName,
						Namespace: TestNamespace,
					},
				}
				scKey := utils.GetResourceKey(sc)
				Expect(k8sClient.Get(ctx, scKey, sc)).Should(Succeed())

				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      TestAddOnParamsSecretName,
						Namespace: TestNamespace,
					},
					Data: map[string][]byte{
						"size": []byte("4"),
					},
				}
				Expect(k8sClient.Update(ctx, secret)).Should(Succeed())

				utils.WaitForResourceSpecToUpdate(k8sClient, ctx, sc, timeout, interval)

				var ds *ocsv1.StorageDeviceSet = nil
				for index := range sc.Spec.StorageDeviceSets {
					item := &sc.Spec.StorageDeviceSets[index]
					if item.Name == deviceSetName {
						ds = item
						break
					}
				}
				Expect(ds.Count == 4).Should(BeTrue())
			})
		})

		When("the sds count is decreased in secret", func() {
			It("should not decrease storage cluster size", func() {
				ctx := context.Background()

				managedOCS := &v1.ManagedOCS{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ManagedOCSName,
						Namespace: TestNamespace,
					},
				}
				managedOCSKey := utils.GetResourceKey(managedOCS)
				Expect(k8sClient.Get(ctx, managedOCSKey, managedOCS)).Should(Succeed())

				sc := &ocsv1.StorageCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      storageClusterName,
						Namespace: TestNamespace,
					},
				}
				scKey := utils.GetResourceKey(sc)
				Expect(k8sClient.Get(ctx, scKey, sc)).Should(Succeed())

				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      TestAddOnParamsSecretName,
						Namespace: TestNamespace,
					},
					Data: map[string][]byte{
						"size": []byte("1"),
					},
				}
				Expect(k8sClient.Update(ctx, secret)).Should(Succeed())

				time.Sleep(3 * time.Second)

				Expect(k8sClient.Get(ctx, scKey, sc)).Should(Succeed())

				var ds *ocsv1.StorageDeviceSet = nil
				for index := range sc.Spec.StorageDeviceSets {
					item := &sc.Spec.StorageDeviceSets[index]
					if item.Name == deviceSetName {
						ds = item
						break
					}
				}
				Expect(ds.Count != 1).Should(BeTrue())
			})
		})

		When("the storage cluster is modfied while not in strict mode", func() {
			It("should not revert any changes back to the managed state", func() {
				ctx := context.Background()

				managedOCS := &v1.ManagedOCS{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ManagedOCSName,
						Namespace: TestNamespace,
					},
				}
				managedOCSKey := utils.GetResourceKey(managedOCS)
				Expect(k8sClient.Get(ctx, managedOCSKey, managedOCS)).Should(Succeed())

				// Change the reconcile strategy to none
				managedOCS.Spec.ReconcileStrategy = v1.ReconcileStrategyNone
				Expect(k8sClient.Update(ctx, managedOCS)).Should(Succeed())
				Eventually(func() bool {
					err := k8sClient.Get(ctx, managedOCSKey, managedOCS)
					return err == nil && managedOCS.Status.ReconcileStrategy == v1.ReconcileStrategyNone
				}, timeout, interval).Should(BeTrue())

				sc := &ocsv1.StorageCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      storageClusterName,
						Namespace: TestNamespace,
					},
				}
				scKey := utils.GetResourceKey(sc)
				Expect(k8sClient.Get(ctx, scKey, sc)).Should(Succeed())

				defaults := ocsv1.StorageClusterSpec{}
				sc.Spec = defaults
				Expect(k8sClient.Update(ctx, sc)).Should(Succeed())
				Consistently(func() bool {
					err := k8sClient.Get(ctx, scKey, sc)
					return err == nil && reflect.DeepEqual(sc.Spec, defaults)
				}, duration, interval).Should(BeTrue())
			})
		})
	})
})

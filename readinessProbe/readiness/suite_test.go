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

package readiness

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	ocsv1 "github.com/openshift/ocs-operator/pkg/apis"
	v1 "github.com/openshift/ocs-osd-deployer/api/v1alpha1"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.
const (
	ManagedOCSName = "test-managedocs"
	TestNamespace  = "default"

	// Define utility constants for object names and testing timeouts/durations and intervals.
	timeout  = time.Second * 10
	duration = time.Second * 10
	interval = time.Millisecond * 250
)

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Readiness Probe Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))

	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "config", "crd", "bases"),
		},
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	// Setup client options
	var options client.Options

	// The readiness must have these schemes to deserialize the k8s objects
	options.Scheme = runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(options.Scheme))
	utilruntime.Must(ocsv1.AddToScheme(options.Scheme))
	utilruntime.Must(v1.AddToScheme(options.Scheme))

	// Client to be use by the test code, using a non cached client
	k8sClient, err = client.New(cfg, options)
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	go RunServer(k8sClient, types.NamespacedName{Name: ManagedOCSName, Namespace: TestNamespace}, ctrl.Log.WithName("readiness"))

	ctx := context.Background()

	managedOCS := &v1.ManagedOCS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ManagedOCSName,
			Namespace: TestNamespace,
		},
	}
	Expect(k8sClient.Create(ctx, managedOCS)).Should(Succeed())

	close(done)
}, 60)

var _ = AfterSuite(func() {
	ctx := context.Background()

	managedOCS := &v1.ManagedOCS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ManagedOCSName,
			Namespace: TestNamespace,
		},
	}
	Expect(k8sClient.Delete(ctx, managedOCS)).Should(Succeed())

	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

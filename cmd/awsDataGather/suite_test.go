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
package main

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"net/http"

	v1 "github.com/red-hat-storage/ocs-osd-deployer/api/v1alpha1"

	"github.com/go-logr/logr"
	// +kubebuilder:scaffold:imports
)

const (
	testNamespace  = "default"
	testMockerAddr = "localhost:8082"
)

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "AWS data gathering Suite")
}

var _ = BeforeSuite(func() {
	done := make(chan interface{})
	go func() {
		logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

		var err error
		testEnv = &envtest.Environment{}
		cfg, err = testEnv.Start()
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg).ToNot(BeNil())

		// Setup client options
		var options client.Options

		options.Scheme = runtime.NewScheme()
		utilruntime.Must(clientgoscheme.AddToScheme(options.Scheme))
		utilruntime.Must(v1.AddToScheme(options.Scheme))

		// Client to be use by the test code, using a non cached client
		k8sClient, err = client.New(cfg, options)
		Expect(err).ToNot(HaveOccurred())
		Expect(k8sClient).ToNot(BeNil())

		go runIMDSv1MockServer(testMockerAddr, ctrl.Log.WithName("IMDSv1Mocker"))

		close(done)
	}()
	Eventually(done, 60).Should(BeClosed())
})

var _ = AfterSuite(func() {
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

const (
	macPath  = "/latest/meta-data/mac"
	cidrPath = "/latest/meta-data/network/interfaces/macs/ff:ff:ff:ff:ff:ff/vpc-ipv4-cidr-blocks"
	fakeMac  = "ff:ff:ff:ff:ff:ff"
	fakeCIDR = "10.0.0.0/16;10.206.54.0/24"
)

func runIMDSv1MockServer(listenAddr string, log logr.Logger) {
	server := &http.Server{Addr: listenAddr}

	http.HandleFunc(macPath, func(writer http.ResponseWriter, req *http.Request) {
		log.Info("Received request for mac address")
		_, err := writer.Write([]byte(fakeMac))
		if err != nil {
			log.Error(err, "Failed to write response body")
			writer.WriteHeader(http.StatusServiceUnavailable)
		}
	})

	http.HandleFunc(cidrPath, func(writer http.ResponseWriter, req *http.Request) {
		log.Info("Received request for vpc cidr")
		_, err := writer.Write([]byte(fakeCIDR))
		if err != nil {
			log.Error(err, "Failed to write response body: %s")
			writer.WriteHeader(http.StatusServiceUnavailable)
		}
	})

	log.Info("Starting fake IMDS v1 server...")
	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Error(err, "Failed to start fake IMDS v1 server")
	}
}

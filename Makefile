# ===== Overridable Variables ===== #

# Current Operator version
VERSION ?= 2.0.11
REPLACES ?= 2.0.10

# Default bundle image tag
IMAGE_TAG_BASE ?= controller
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:v$(VERSION)

# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

OUTPUT_DIR ?= bundle

# Image URL to use all building/pushing image targets
IMG ?= ocs-osd-deployer:latest

# USE_IMAGE_DIGESTS defines if images are resolved via tags or digests
USE_IMAGE_DIGESTS ?= false
ifeq ($(USE_IMAGE_DIGESTS), true)
    BUNDLE_METADATA_OPTS += --use-image-digests
endif

BUNDLE_IMGS ?= $(BUNDLE_IMG)
CATALOG_IMG ?= $(IMAGE_TAG_BASE)-catalog:$(VERSION)

ifneq ($(origin CATALOG_BASE_IMG), undefined)
	FROM_INDEX_OPT := --from-index $(CATALOG_BASE_IMG)
endif

LOCALBIN ?= $(CURDIR)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
OPERATOR_SDK = $(LOCALBIN)/operator-sdk
OPM ?= $(LOCALBIN)/opm

## Tool Versions
KUSTOMIZE_VERSION ?= v3.9.1
CONTROLLER_TOOLS_VERSION ?= v0.8.0
ENVTEST_VERSION ?= latest
OPERATOR_SDK_VERSION ?= v1.21.0
OPM_VERSION ?= v1.19.1

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"

# ===== Constants ===== #

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.23

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

OS = $(shell go env GOOS)
ARCH = $(shell go env GOARCH)

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

ifndef ignore-not-found
  ignore-not-found = false
endif

# ===== Make targets ===== #

.PHONY: all
all: manager readinessServer awsDataGather

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test-converged
test-converged: manifests generate fmt vet envtest ## Run converged mode tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" DEPLOYMENT_TYPE=converged go test ./... -coverprofile cover.out

.PHONY: test-provider
test-provider: manifests generate fmt vet envtest ## Run provider mode tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" DEPLOYMENT_TYPE=provider go test ./... -coverprofile cover.out

.PHONY: test-consumer
test-consumer: manifests generate fmt vet envtest ## Run consumer mode tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" DEPLOYMENT_TYPE=consumer go test ./... -coverprofile cover.out

.PHONY: test
test: test-converged test-provider test-consumer ## Run tests.

##@ Build

.PHONY: manager
manager: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

.PHONY: readinessServer
readinessServer: fmt vet ## Build readiness probe binary.
	go build -o bin/readinessServer readinessProbe/main.go

.PHONY: awsDataGather
awsDataGather: cmd/awsDataGather/main.go pkg/aws/imds_client.go
	go build -o bin/awsDataGather cmd/awsDataGather/main.go

.PHONY: export_env_vars
export_env_vars:
export NAMESPACE = openshift-storage
export ADDON_NAME = ocs-converged
export SOP_ENDPOINT = https://red-hat-storage.github.io/ocs-sop/sop/OSD/{{ .GroupLabels.alertname }}.html
export ALERT_SMTP_FROM_ADDR = noreply-test@test.com
export DEPLOYMENT_TYPE = converged
export RHOBS_ENDPOINT = https://rhobs.com/fake_url
export RH_SSO_TOKEN_ENDPOINT = https://sso.com/fake_url

.PHONY: run
run: generate fmt vet manifests export_env_vars ## Run a controller from your host.
	kubectl create namespace ${NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -
	kubectl create secret generic addon-${ADDON_NAME}-parameters -n ${NAMESPACE} --from-literal size=1 --from-literal enable-mcg=false --dry-run=client -oyaml | kubectl apply -f -
	kubectl create secret generic ${ADDON_NAME}-pagerduty -n ${NAMESPACE} --from-literal PAGERDUTY_KEY="test-key" --dry-run=client -oyaml | kubectl apply -f -
	kubectl create secret generic ${ADDON_NAME}-deadmanssnitch -n ${NAMESPACE} --from-literal SNITCH_URL="https://test-url" --dry-run=client -oyaml | kubectl apply -f -
	kubectl create secret generic ${ADDON_NAME}-smtp -n ${NAMESPACE} --from-literal host="smtp.sendgrid.net" --from-literal password="test-key" --from-literal port="587" \
	--from-literal username="apikey" --dry-run=client -oyaml | kubectl apply -f -
	kubectl create configmap rook-ceph-operator-config -n ${NAMESPACE} --dry-run=client -oyaml | kubectl apply -f -
	envsubst < makefileutils.yaml | kubectl apply -f -
	go run ./main.go

.PHONY: docker-build
docker-build: test ## Build docker image with the manager.
	docker build . -t ${IMG}

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

##@ Deployment

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -
	./shim/shim.sh install

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -
	./shim/shim.sh uninstall

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: bundle
bundle: manifests kustomize operator-sdk ## Generate bundle manifests and metadata, then validate generated files.
	$(OPERATOR_SDK) generate kustomize manifests --interactive=false -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	cd config/aws-data-gather && $(KUSTOMIZE) edit set image controller=$(IMG)
	cd config/manifests/bases && \
		rm -rf kustomization.yaml && \
		$(KUSTOMIZE) create --resources ocs-osd-deployer.clusterserviceversion.yaml && \
		$(KUSTOMIZE) edit add annotation --force 'olm.skipRange':">=0.0.1 <$(VERSION)" && \
		$(KUSTOMIZE) edit add patch --name ocs-osd-deployer.v0.0.0 --kind ClusterServiceVersion \
		--patch '[{"op": "replace", "path": "/spec/replaces", "value": "ocs-osd-deployer.v$(REPLACES)"}]'
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle \
		-q \
		--extra-service-accounts prometheus-k8s,aws-data-gather \
		--overwrite \
		--version $(VERSION) \
		$(BUNDLE_METADATA_OPTS) \
		--output-dir=$(OUTPUT_DIR)
	cp config/metadata/* $(OUTPUT_DIR)/metadata/
	$(OPERATOR_SDK) bundle validate $(OUTPUT_DIR)

.PHONY: bundle-build
bundle-build: ## Build the bundle image.
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	$(MAKE) docker-push IMG=$(BUNDLE_IMG)

.PHONY: catalog-build
catalog-build: opm ## Build the catalog image.
	$(OPM) index add --container-tool docker --mode semver --tag $(CATALOG_IMG) --bundles $(BUNDLE_IMGS) $(FROM_INDEX_OPT)

.PHONY: catalog-push
catalog-push: ## Push the catalog image.
	$(MAKE) docker-push IMG=$(CATALOG_IMG)

##@ Build Dependencies

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
ifeq (,$(wildcard $(KUSTOMIZE)))
$(KUSTOMIZE): $(LOCALBIN)
	curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN)
endif

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
ifeq (,$(wildcard $(CONTROLLER_GEN)))
$(CONTROLLER_GEN): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)
endif

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
ifeq (,$(wildcard $(ENVTEST)))
$(ENVTEST): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@$(ENVTEST_VERSION)
endif

.PHONY: operator-sdk
operator-sdk: $(OPERATOR_SDK) ## Download operator-sdk locally if necessary
ifeq (,$(wildcard $(OPERATOR_SDK)))
$(OPERATOR_SDK): $(LOCALBIN)
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPERATOR_SDK)) ;\
	curl -sSLo $(OPERATOR_SDK) https://github.com/operator-framework/operator-sdk/releases/download/$(OPERATOR_SDK_VERSION)/operator-sdk_${OS}_${ARCH};\
	chmod +x $(OPERATOR_SDK) ;\
	}
endif

.PHONY: opm
opm: $(OPM) ## Download opm locally if necessary.
ifeq (,$(wildcard $(OPM)))
$(OPM): $(LOCALBIN)
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/$(OPM_VERSION)/$(OS)-$(ARCH)-opm ;\
	chmod +x $(OPM) ;\
	}
endif

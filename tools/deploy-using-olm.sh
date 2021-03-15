#!/bin/bash

# tools definitions
K8S_CLIENT=${K8S_CLIENT:-kubectl}
OP_SDK=${OP_SDK:-operator-sdk}

# Bundle definition
BUNDLE_REPO=${BUNDLE_REPO:-${1}}
if [[ -z ${BUNDLE_REPO} ]]; 
then
   echo "Error: target repository was not set via a command line argument or the BUNDLE_REPO environment variable"
   exit 1
fi
BUNDLE_NAME=${BUNDLE_NAME:-ocs-osd-deployer-bundle}
BUNDLE_VERSION=${BUNDLE_VERSION:-latest}
BUNDLE_IMAGE=${BUNDLE_IMAGE:-${BUNDLE_REPO}/${BUNDLE_NAME}:${BUNDLE_VERSION}}

# Deploy target
TARGET_NAMESPACE=${TARGET_NAMESPACE:-openshift-storage}

echo "Generate deployer image"
make docker-build

# Generate the olm bundle image
echo "Generate and push the olm bundle image: ${BUNDLE_IMAGE}"
make manifests
make bundle
make bundle-build
docker tag controller-bundle:0.0.1 ${BUNDLE_IMAGE}
docker push ${BUNDLE_IMAGE}

echo "Checking for existing olm"
${K8S_CLIENT} get namespace olm >/dev/null 2>&1
if [ $? == 0 ]; 
then
    echo "olm found"
else 
    echo "olm not found, installing olm in target cluster"
    ${OP_SDK} olm install     
fi

echo "Deploy/Run the ${BUNDLE_NAME}:${BUNDLE_VERSION} in the cluster (at ${TARGET_NAMESPACE})"
${K8S_CLIENT} create namespace ${TARGET_NAMESPACE}
${K8S_CLIENT} create -f ./tools/ocs-catalogsource.yaml
${OP_SDK} run bundle ${BUNDLE_IMAGE} -n ${TARGET_NAMESPACE}

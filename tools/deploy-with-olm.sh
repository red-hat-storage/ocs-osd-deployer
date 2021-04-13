#!/bin/bash

# enable !! command completion
set -o history -o histexpand

source "$(dirname ${0})/colors.sh"

# Allow controlled exit on error
function exit_on_err() {
    "$@"
    EXIT_CODE=$?
    if [ ${EXIT_CODE} -ne 0 ]; then
        >&2 echo "\"${@}\" command failed with exit code ${EXIT_CODE}."
        exit ${EXIT_CODE}
    fi
}

# tools definitions
K8S_CLIENT=${K8S_CLIENT:-kubectl}
OP_SDK=${OP_SDK:-operator-sdk}

# Bundle definition
IMAGE_REPO=${IMAGE_REPO:-${1}}
if [[ -z ${IMAGE_REPO} ]]; then
   red "Error: target repository was not set via a command line argument or the IMAGE_REPO environment variable"
   exit 1
fi

# Bundle image
BUNDLE_NAME=${BUNDLE_NAME:-ocs-osd-deployer-bundle}
BUNDLE_VERSION=${BUNDLE_VERSION:-latest}
BUNDLE_IMAGE=${BUNDLE_IMAGE:-${IMAGE_REPO}/${BUNDLE_NAME}:${BUNDLE_VERSION}}

# Deployer image
DEPLOYER_NAME=${DEPLOYER_NAME:-ocs-osd-deployer}
DEPLOYER_VERSION=${DEPLOYER_VERSION:-latest}
DEPLOYER_IMAGE=${DEPLOYER_IMAGE:-${IMAGE_REPO}/${DEPLOYER_NAME}:${DEPLOYER_VERSION}}

# Deploy target
TARGET_NAMESPACE=${TARGET_NAMESPACE:-openshift-storage}

# Generate the deployer image 
blue "Generate deployer image: ${DEPLOYER_IMAGE}"
exit_on_err make docker-build IMG=${DEPLOYER_IMAGE}
exit_on_err make docker-push IMG=${DEPLOYER_IMAGE}

# Generate the olm bundle image
blue "Generate and push the olm bundle image: ${BUNDLE_IMAGE}"
exit_on_err make bundle IMG=${DEPLOYER_IMAGE}
exit_on_err make bundle-build BUNDLE_IMG=${BUNDLE_IMAGE}
exit_on_err make docker-push IMG=${BUNDLE_IMAGE}

# In install olm if needed
blue "Checking for existing olm"
${K8S_CLIENT} get namespace olm >/dev/null 2>&1
if [ $? -eq 0 ]; then
    blue "olm found"
else 
    blue "olm not found, installing olm in target cluster"
    exit_on_err ${OP_SDK} olm install     
fi

# Deploy the bundle
blue "Deploy/Run the ${BUNDLE_NAME}:${BUNDLE_VERSION} in the cluster (at ${TARGET_NAMESPACE})"
${K8S_CLIENT} get namespace olm >/dev/null 2>&1
if [ $? -eq 0 ]; then
    exit_on_err ${K8S_CLIENT} create namespace ${TARGET_NAMESPACE}
fi
exit_on_err ${K8S_CLIENT} apply -f ./tools/ocs-catalogsource.yaml
exit_on_err ${OP_SDK} run bundle ${BUNDLE_IMAGE} -n ${TARGET_NAMESPACE}

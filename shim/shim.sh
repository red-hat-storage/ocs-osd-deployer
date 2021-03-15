#!/bin/bash
 
SELF=$(basename ${0})
ARGS=$@
VERBOSE="off"
CRD_KIND="customresourcedefinitions.apiextensions.k8s.io"

K8S_CLIENT=kubectl
ROOT_DIR="$(cd "$(dirname "$0")/.." >/dev/null 2>&1 && pwd)"
CRDS_DIR="${ROOT_DIR}/shim/crds"
CRDS=($(ls ${CRDS_DIR}))

function usage() {
  echo -e "Usage: ${SELF} action [--verbose|-v]"
  echo -e "Actions:"
  echo -e "  install:\tInstalls shim CRDs into a cluster"
  echo -e "  uninstall:\tUninstalls shim CRDs from a cluster"
  echo -e
  echo -e "Options:"
  echo -e "  -v or --verbose:\tPrint a verbose output including the ${K8S_CLIENT} command issued and their output"
  echo -e
}

function install() {
  for CRD_FILE in "${CRDS[@]}"; do
    CRD=${CRD_FILE%.*}
    _run_command \
      "${K8S_CLIENT} create -f ${CRDS_DIR}/${CRD_FILE}" \
      "${CRD_KIND}/${CRD} created" \
      "${CRD_KIND}/${CRD} already exists on target cluster"
  done
}

function uninstall() {
  for CRD_FILE in ${CRDS[@]}; do
    CRD=${CRD_FILE%.*}
    _run_command \
      "${K8S_CLIENT} delete -f ${CRDS_DIR}/${CRD_FILE}" \
      "${CRD_KIND}/${CRD} deleted" \
      "${CRD_KIND}/${CRD} does not exists on target cluster"
  done
}

function _run_command() {
  COMMAND=${1}
  SUCCESS=${2}
  FAILURE=${3}
  
  # Handle verbose output 
  if [[ ${VERBOSE} == "on" ]]; then 
    OUTPUT=/dev/stdout
    echo -e "[$(date)] ${COMMAND}"
  else 
    OUTPUT=/dev/null
  fi

  # Run the command
  ${COMMAND} > ${OUTPUT} 2>&1
  if [ $? == 0 ]; then
      echo ${SUCCESS}
  else 
      echo ${FAILURE}
  fi
}

function main() {
  ACTION=""

  for ARG in ${ARGS[@]}; do 
    case ${ARG} in
      install)
        if [[ -z ${ACTION} ]]; then 
          ACTION="install";
        else 
          ACTION="usage"
        fi
        ;;

      uninstall)
        if [[ -z ${ACTION} ]]; then 
          ACTION="uninstall";
        else 
          ACTION="usage"
        fi
        ;;

      --verbose | -v)
        if [[ ${VERBOSE} == "off" ]]; then 
          VERBOSE="on"
        else 
          ACTION="usage"
        fi
        ;;

      *)
        ACTION="usage"
        ;;
    esac
  done 



  case ${ACTION} in 
    install) install;;
    uninstall) uninstall;;
    *) usage;;
  esac  
}

main




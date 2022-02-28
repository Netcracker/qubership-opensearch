#!/usr/bin/env bash

#Required for DP job to obtain parameters
eval $(sed -e 's/:[^:\/\/]/="/g;s/$/"/g;s/ *=/=/g' <<< "$DEPLOYMENT_PARAMETERS" | grep "ENABLE_MIGRATION\|ENABLE_HELM_DELETE\|ENABLE_PV_PATCH\|PREVIOUS_CUSTOM_RESOURCE_NAME\|PREVIOUS_DEDICATED_ARBITER\|PREVIOUS_DEDICATED_DATA")

ENABLE_MIGRATION=${ENABLE_MIGRATION:-false}
ENABLE_HELM_DELETE=${ENABLE_HELM_DELETE:-true}
ENABLE_PV_PATCH=${ENABLE_PV_PATCH:-true}

PREVIOUS_CUSTOM_RESOURCE_NAME=${PREVIOUS_CUSTOM_RESOURCE_NAME:-elasticsearch}

PREVIOUS_DEDICATED_ARBITER=${PREVIOUS_DEDICATED_ARBITER:-false}
PREVIOUS_DEDICATED_DATA=${PREVIOUS_DEDICATED_MASTER:-false}

echo "ENABLE_MIGRATION: ${ENABLE_MIGRATION}"
if [[ ${ENABLE_MIGRATION} != "true" ]]; then
    exit 0
fi

#$1- pvc prefix
migrate_pvc() {
  local pvc_prefix=$1

  pvc_names=($($kubectl get pvc --no-headers -o custom-columns=:.metadata.name | grep "${pvc_prefix}-[0-9]"))

  for pvc_name in "${pvc_names[@]}"; do
    echo "Migrate PVC: $pvc_name"
    pv_name=$($kubectl get pvc "${pvc_name}" -o jsonpath='{.spec.volumeName}')
    if [[ $? -ne 0 ]]; then
      echo "Error: failed to get PV name from PVC $pvc_name. Please perform manual migration"
      exit 0
    fi
    echo "Migrate PV: $pv_name"

    $kubectl patch pv "${pv_name}" -p '{"spec":{"persistentVolumeReclaimPolicy":"Retain"}}'
    if [[ $? -ne 0 ]]; then
      echo "Error: failed to patch PV $pv_name. Please perform manual migration"
      exit 0
    fi
    $kubectl delete pvc "${pvc_name}"
    if [[ $? -ne 0 ]]; then
      echo "Error: failed to delete PVC $pvc_name. Please perform manual migration"
      exit 0
    fi
    $kubectl patch pv "${pv_name}" -p '{"spec":{"claimRef": null}}'
    if [[ $? -ne 0 ]]; then
      echo "Error: failed to patch PV $pv_name. Please perform manual migration"
      exit 0
    fi
  done
}

if command -v kubectl &> /dev/null; then
    kubectl="kubectl"
else
    source ${WORKSPACE}/oc_version_used.sh
    kubectl="${OCBINVERP}"
fi

if command -v helm &> /dev/null; then
    helm="helm"
  else
    helm="helm3"
  fi

echo "Start migration procedure"

if ! ($helm list | grep elasticsearch-service) ; then
    echo "There are no elasticsearch-service helm releases. Please perform manual migration"
    exit 0
fi

if [[ ${ENABLE_HELM_DELETE} == "true" ]]; then
  $helm uninstall elasticsearch-service
  $helm uninstall elasticsearch-service-${NAMESPACE}
  echo "releases has been deleted, wait 10s for terminating resources"
  sleep 10
fi

if [[ ${ENABLE_PV_PATCH} != "true" ]]; then
  echo "End migration procedure"
  exit 0
fi

echo "Start patching PV"
master_pvc_prefix="pvc-${PREVIOUS_CUSTOM_RESOURCE_NAME}"
if [[ ${PREVIOUS_DEDICATED_DATA} == "true" ]]; then
    master_pvc_prefix="${master_pvc_prefix}-master"
fi
migrate_pvc ${master_pvc_prefix}

if [[ ${PREVIOUS_DEDICATED_DATA} == "true" ]]; then
    data_pvc_prefix="pvc-${PREVIOUS_CUSTOM_RESOURCE_NAME}-data"
    migrate_pvc ${master_pvc_prefix}
fi

if [[ ${PREVIOUS_DEDICATED_DATA} == "true" ]]; then
    data_pvc_prefix="pvc-${PREVIOUS_CUSTOM_RESOURCE_NAME}-data"
    migrate_pvc ${data_pvc_prefix}
fi

if [[ ${PREVIOUS_DEDICATED_ARBITER} == "true" ]]; then
    arbiter_pvc_prefix="pvc-${PREVIOUS_CUSTOM_RESOURCE_NAME}-arbiter"
    migrate_pvc ${arbiter_pvc_prefix}
fi

echo "End migration procedure"
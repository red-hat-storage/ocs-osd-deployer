set -e

# Path to the custom rule file.
RULE_FILE="controllers/managed-ocs-rules.yaml"

# Unwanted fields in the rule file.
declare -a UNWANTED_FIELDS=("oid" "documentation")

# Unwanted alerts in the rule file.
declare -a UNWANTED_ALERTS=() # TODO: Alerts in Ceph that are not relevant to `managed-ocs` will go here.

# Fetch the latest Ceph rule file.
curl -SsL https://raw.githubusercontent.com/ceph/ceph/main/monitoring/ceph-mixin/prometheus_alerts.yml --output "${RULE_FILE}"

# Remove unwanted fields from the rule file.
for rule in "${UNWANTED_FIELDS[@]}"
do
  sed -i "/.*${rule}:.*/d" "${RULE_FILE}"
done

# Remove unwanted alerts from the rule file.
for unwanted_alert in "${UNWANTED_ALERTS[@]}"
do
  GROUP="${unwanted_alert%/*}"
  ALERT="${unwanted_alert#*/}"
  echo "Removing ${ALERT} from ${GROUP} group."
  yq "del(.groups.[] | select(.name == \"${GROUP}\").rules[] | select(.alert == \"${ALERT}\"))" "${RULE_FILE}" > out.yaml
  cat out.yaml > "${RULE_FILE}"
  rm -f out.yaml
done

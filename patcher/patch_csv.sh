#! /bin/bash

set -e -o pipefail

function usage {
    cat - <<USAGE
Usage:
    $0 <OCS_CATALOG_IMAGE> <DEST_DIR>

    Extracts the CSVs and CRDs from the catalog into DEST_DIR and patches them
    to include the deployer.
USAGE
}

if [[ $# -ne 2 ]]; then
    usage
    exit 1
fi

SCRIPTDIR="$(dirname "$(realpath "$0")")"
IMAGE="$1"
DEST="$2"

mkdir -p "$DEST"
cd "$DEST"
docker run --rm --entrypoint /bin/bash "$IMAGE" -c "tar cf - -C /manifests ." | tar xf -

while IFS= read -r -d '' csv
do
    echo "Patching $csv ..."
    # yq m -a "$csv" "$SCRIPTDIR/csv_fragment.yml" > "$csv.j2"
    # rm -f "$csv"
    yq m -a -i "$csv" "$SCRIPTDIR/csv_fragment.yml"
done <   <(find . -name '*clusterserviceversion.yaml' -print0)

echo
echo "=============="
echo "= Update the addon.yaml with the following information:"
echo
cat ocs-operator.package.yaml
rm -f ocs-operator.package.yaml

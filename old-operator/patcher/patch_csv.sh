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

function finish {
  docker rm -f dummy > /dev/null 2>&1
}
trap finish EXIT


if [[ $# -ne 2 ]]; then
    usage
    exit 1
fi

SCRIPTDIR="$(dirname "$(realpath "$0")")"
IMAGE="$1"
DEST="$2"

mkdir -p "$DEST"
cd "$DEST"


#docker run --rm --entrypoint /bin/bash "$IMAGE" -c "tar cf - -C /manifests ." | tar xf -

#Bundle images do not have a runtime so the "docker run" command fails

docker create -ti --name dummy "$IMAGE" bash
docker cp dummy:/manifests .
docker rm -f dummy


while IFS= read -r -d '' csv
do
    echo "Patching $csv ..."
    # yq m -a "$csv" "$SCRIPTDIR/csv_fragment.yml" > "$csv.j2"
    # rm -f "$csv"
    yq m -a append -i "$csv" "$SCRIPTDIR/csv_fragment.yml"
done <   <(find . -name '*clusterserviceversion.yaml' -print0)


#This file is not available in the bundle image

if [ -f "ocs-operator.package.yaml" ]; then
	echo
	echo "=============="
	echo "= Update the addon.yaml with the following information:"
	echo
	cat ocs-operator.package.yaml
	rm -f ocs-operator.package.yaml
fi

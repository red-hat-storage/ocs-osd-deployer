# OCS deployer patch utilies

These scripts assist in creating the files to update the managed tenants repo w/
the catalog information to deploy OCS.

## `patch_csv.sh <catalog_image> <output_dir>`

This script extracts the manifests from the catalog image into the provided
output directory and patches the CSVs to include the deployer.

**Important Notes:**

* This script requires `yq`. [Get it here.](https://github.com/mikefarah/yq)
* You may need to manually pull the catalog image before running the script
  since they require authentication to pull.

Example usage:

```console
$ ./patch_csv.sh quay.io/rhceph-dev/ocs-olm-operator:latest-4.4 out
Updating ./4.2.2/ocs-operator.v4.2.2.clusterserviceversion.yaml ...
Updating ./4.3.0/ocs-operator.v4.3.0.clusterserviceversion.yaml ...
Updating ./4.4.0-437.ci/ocs-operator.v4.4.0-437.ci.clusterserviceversion.yaml ...
Updating ./4.2.1/ocs-operator.v4.2.1.clusterserviceversion.yaml ...
Updating ./4.2.3/ocs-operator.v4.2.3.clusterserviceversion.yaml ...
Updating ./4.2.0/ocs-operator.v4.2.0.clusterserviceversion.yaml ...

==============
= Update the addon.yaml with the following information:

packageName: ocs-operator
defaultChannel: stable-4.4
channels:
- name: stable-4.2
  currentCSV: ocs-operator.v4.2.3
- name: stable-4.3
  currentCSV: ocs-operator.v4.3.0
- name: stable-4.4
  currentCSV: ocs-operator.v4.4.0-437.ci


$ ls out/4.4.0-437.ci/
./                            noobaa.crd.yaml
../                           noobaa-metrics-role_binding.yaml
backingstore.crd.yaml         noobaa-metrics-role.yaml
bucketclass.crd.yaml          ocsinitialization.crd.yaml
cephblockpool.crd.yaml        ocs-operator.v4.4.0-437.ci.clusterserviceversion.yaml.j2
cephclient.crd.yaml           rook-metrics-role_binding.yaml
cephcluster.crd.yaml          rook-metrics-role.yaml
cephfilesystem.crd.yaml       rook-monitor-role_binding.yaml
cephnfs.crd.yaml              rook-monitor-role.yaml
cephobjectstore.crd.yaml      storagecluster.crd.yaml
cephobjectstoreuser.crd.yaml  storageclusterinitialization.crd.yaml
```

The CSV has been converted to a Jinja template and the deployer has been added
to it.

These patched manifests can now be committed to the managed tenants repo in the
`bundles` directory.

The OCS addon.yaml file in the managed-tenants repo for the appropriate
environment should be updated thus:

```yaml
...
# Update channels and defaultChannel based on latest CSV
channels:
- currentCSV: ocs-operator.v4.4.0-437.ci
  name: stable-4.4
defaultChannel: stable-4.4
...
```

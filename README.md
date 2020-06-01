# OCS on OSD deployer

This repo contains the code necessary to deploy OCS on OpenShift Dedicated as an
"addon".

## Components

* [`deployer/`](deployer) - Contains the source to build the "deployer"
  container. Its purpose is to automate the deployment of the storage cluster
  based on the addon configuration passed in (eventually) from OCM.
* [`patcher/`](patcher) - Contains the code to patch the standard OCS operator
  CSVs to add the deployer Deployment object & associated RBAC.

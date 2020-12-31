/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package templates

// StorageClusterTemplate is the template that serves as the base for the storage clsuter deployed by the operator
const StorageClusterTemplate = `
apiVersion: ocs.openshift.io/v1
kind: StorageCluster
spec:
  # The empty label selector removes the default so components can run an all
  # worker nodes.
  labelSelector:
    matchExpressions: []
  manageNodes: false
  monPVCTemplate:
    spec:
      storageClassName: gp2
      accessModes:
        - ReadWriteOnce
  resources:
    mds:
      limits:
        cpu: 3000m
        memory: 8Gi
      requests:
        cpu: 1000m
        memory: 8Gi
    mgr:
      limits:
        cpu: 1000m
        memory: 3Gi
      requests:
        cpu: 1000m
        memory: 3Gi
    mon:
      limits:
        cpu: 1000m
        memory: 2Gi
      requests:
        cpu: 1000m
        memory: 2Gi
    noobaa-core:
      limits: {}
      requests: {}
    noobaa-db:
      limits: {}
      requests: {}
  storageDeviceSets:
    - name: mydeviceset
      count: 3 
      dataPVCTemplate:
        spec:
          storageClassName: gp2
          accessModes:
            - ReadWriteOnce
          volumeMode: Block
          resources:
            requests:
              storage: 1000Gi
      placement: {}
      portable: true
      resources:
        limits:
          cpu: 2000m
          memory: 5Gi
        requests:
          cpu: 1000m
          memory: 5Gi
`

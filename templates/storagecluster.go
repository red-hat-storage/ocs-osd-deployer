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
  # The label selector is used to select only the worker nodes for
  # both labeling and scheduling.
  labelSelector:
    matchExpressions:
      - key: node-role.kubernetes.io/worker
        operator: Exists
      - key: node-role.kubernetes.io/infra
        operator: DoesNotExist
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
  storageDeviceSets:
    - name: default
      count: 1
      dataPVCTemplate:
        spec:
          storageClassName: gp2
          accessModes:
            - ReadWriteOnce
          volumeMode: Block
          resources:
            requests:
              storage: 1Ti
      placement: {}
      portable: true
      replica: 3
      resources:
        limits:
          cpu: 2000m
          memory: 5Gi
        requests:
          cpu: 1000m
          memory: 5Gi
  multiCloudGateway:
    reconcileStrategy: "ignore"
`

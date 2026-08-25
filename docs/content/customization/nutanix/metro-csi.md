+++
title = "Metro CSI computeAffinity"
+++

Metro clusters span two Prism Elements. Nutanix CSI `computeAffinity` values other than
`DISABLED` pin a volume to a single Prism Element and are not supported on metro.

When a Cluster uses a `NutanixMetro/` or `NutanixMetroSite/` failure domain, Cluster
admission:

- Sets `computeAffinity: DISABLED` on every Nutanix CSI StorageClassConfig that does not
  already set the parameter.
- Rejects the Cluster if `computeAffinity` is set to any other value, with an error that
  it is not supported for the metro cluster.

## Example

```yaml
apiVersion: cluster.x-k8s.io/v1beta1
kind: Cluster
metadata:
  name: <NAME>
spec:
  topology:
    variables:
      - name: clusterConfig
        value:
          addons:
            csi:
              providers:
                nutanix:
                  storageClassConfigs:
                    volume:
                      parameters:
                        storageContainer: <STORAGE_CONTAINER>
                        computeAffinity: DISABLED
```

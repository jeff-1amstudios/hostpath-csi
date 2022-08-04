
# hostpath-csi

CSI driver that supports ephemeral hostpath volumes

## Deployment
A Helm chart provided in chart/dummy-fuse-csi may be used to deploy the dummy-fuse-csi Node Plugin in the cluster. For Helm v3 use the following command:

```sh
helm install <deployment name> chart/dummy-fuse-csi
```

```sh
kubectl create -f manifests/pod.yaml
```
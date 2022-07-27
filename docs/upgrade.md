# Upgrades

Upgrading the Knative operator will automatically trigger the upgrade of any
existing `KnativeServing` and `KnativeEventing` instances, so you may want to
create backups of them first:

```
kubectl get knativeserving --all-namespaces -oyaml >knativeserving.yaml
kubectl get knativeeventing --all-namespaces -oyaml >knativeeventing.yaml
```

Once you've created those backups, simply apply the new version of the operator
and your knative upgrade will begin immediately.

If something goes wrong, you should re-apply the previous version of the
operator, and then re-apply the backup files.

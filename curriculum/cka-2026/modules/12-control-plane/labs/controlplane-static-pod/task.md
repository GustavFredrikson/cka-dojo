# Create a static Pod on the control-plane node

On node `cp1`, create `/etc/kubernetes/manifests/dojo-static.yaml` defining a
static Pod with:

- name `dojo-static` in namespace `default`;
- image `busybox:1.36`;
- command `sleep 3600`;
- label `dojo.cka/static=exam`.

Confirm that its mirror Pod appears on node `cp1`.

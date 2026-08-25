# Worked solution

```bash
kubectl apply -f - <<'EOF'
apiVersion: v1
kind: Pod
metadata:
  name: report
  namespace: dojo-scheduling-build
spec:
  nodeSelector:
    dojo-tier: special
  containers:
    - name: report
      image: nginx:1.27-alpine
EOF

kubectl -n dojo-scheduling-build get pod report -o wide
```

The scheduler binds the Pod to the only node whose labels satisfy every entry
in `nodeSelector`.

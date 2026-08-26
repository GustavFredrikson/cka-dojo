# Correct a RoleBinding subject

ServiceAccount `builder` in namespace `dojo-rbac-guided` should read Pods.
Role `pod-reader` is correct, but RoleBinding `builder-pods` names the wrong
ServiceAccount subject. Correct the binding without broadening the Role.

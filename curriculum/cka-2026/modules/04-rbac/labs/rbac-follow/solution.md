# Worked solution

The checkpoint commands create ServiceAccount `auditor`, Role `pod-reader` and
RoleBinding `auditor-pods`. Verify positive and negative permissions with
`kubectl auth can-i`; a narrow reader must not delete Pods.

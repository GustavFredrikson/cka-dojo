# Read the cluster's certificates

A kubeadm cluster runs on about a dozen certificates, and one of them
expiring is a failure mode you will meet in the wild long before you meet it
on an exam. This exercise is about knowing where they are and how to read
them -- nothing here is broken.

Work on `cp1`. You will need `kubeadm`, and `openssl x509` for the parts
`kubeadm` does not show you.

Answer each checkpoint with `dojo check <answer>`.

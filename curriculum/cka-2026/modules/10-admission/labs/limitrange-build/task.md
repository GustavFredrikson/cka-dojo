# Give a namespace sensible defaults

Teams keep deploying Pods with no `resources` block into namespace
`dojo-limitrange-build`. Rather than chase them, make the namespace supply the
numbers.

1. Create LimitRange `defaults` in that namespace so every **container**
   created without its own values receives:
   - a request of `100m` CPU and `64Mi` memory
   - a limit of `250m` CPU and `128Mi` memory

2. Then create Pod `probe` — image `busybox:1.36`, command
   `sleep 3600` — writing **no** `resources` block of your own.

The running Pod must end up carrying all four values.

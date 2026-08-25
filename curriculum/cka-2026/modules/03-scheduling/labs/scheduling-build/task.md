# Place a Pod using a node label

Nodes are labelled as follows:

```text
worker1  dojo-tier=general
worker2  dojo-tier=special
```

In namespace `dojo-scheduling-build`, create Pod `report` using image
`nginx:1.27-alpine`. Configure a node selector for `dojo-tier=special` so the
scheduler places it on `worker2`.

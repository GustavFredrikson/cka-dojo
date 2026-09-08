# Make room for the revenue path

Switch to the `dojo-priority` namespace before you start.

Node `worker1` advertises a custom resource, `dojo.cka/slots`, with a capacity
of **2**. Deployment `batch` currently holds both, and no other node has any.

PriorityClass `dojo-critical` exists but was created with the wrong value:
`100`, which is *below* `dojo-batch`. It is supposed to be `100000`.

Do three things:

1. Give `dojo-critical` the value `100000`.
2. Create Deployment `payments` in `dojo-priority`: one replica, image
   `busybox:1.36`, command `sleep 3600`, priority class `dojo-critical`, with
   a limit of `1` on `dojo.cka/slots`.
3. Get it running.

You may not scale, delete or edit `batch`. The scheduler is expected to make
room on its own.

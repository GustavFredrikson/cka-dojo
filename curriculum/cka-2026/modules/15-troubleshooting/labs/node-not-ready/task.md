# A worker has stopped reporting in

One of the worker nodes is no longer in a usable state, and the scheduler has
stopped placing work on it.

Find out why and put the node back into service, so that it reports `Ready`
and can accept new Pods again.

Whatever you fix should survive a reboot of that node.

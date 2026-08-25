# See a taint block and a toleration permit

This follow-along exercise starts with Pod `web` Pending in namespace
`dojo-scheduling-follow`. It is deliberately pinned to `worker2`, which has a
taint the Pod does not tolerate.

You will inspect the scheduler's evidence and then add the exact toleration.

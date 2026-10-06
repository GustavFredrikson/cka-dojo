# One control plane is down

The cluster is serving normally. `kubectl` works, the `ledger` Deployment in
`dojo-ha-lost-member` is running, and nothing is obviously wrong.

But one of the three control planes has stopped taking part.

Find it and bring it back, then prove etcd has all three of its members
answering again.

Do not remove the member and re-add it — it has not been evicted, it has stopped
participating. Putting it back should not require rebuilding anything.

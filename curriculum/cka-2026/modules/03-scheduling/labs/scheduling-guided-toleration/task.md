# Add a known missing toleration

Pod `batch` in namespace `dojo-scheduling-guided` must run on `worker2`. The
node has taint `dedicated=training:NoSchedule`, and the standalone Pod is
missing the corresponding toleration.

Replace the Pod while preserving its name, image and node selector. Add the
matching toleration so the scheduler can bind it to `worker2`.

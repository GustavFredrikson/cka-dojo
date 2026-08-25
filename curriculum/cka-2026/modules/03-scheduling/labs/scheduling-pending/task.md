# A required worker never receives its Pod

Pod `report` in namespace `dojo-scheduling-pending` remains Pending. It is a
standalone Pod and must run on `worker2` using image `nginx:1.27-alpine`.

Investigate the scheduler's evidence and restore the required placement. Keep
the Pod name, image and destination node unchanged.

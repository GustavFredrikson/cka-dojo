# A Service that does not answer

The `shop` namespace runs a small web application behind a ClusterIP Service
called `web`.

The Pods are healthy. Requests sent to the Service do not reach them.

Restore connectivity through the existing Service `web`, on port 80.

Constraints:

- Keep the Service named `web` in namespace `shop`, of type ClusterIP, serving
  on port 80.
- Do not delete and recreate the Deployment.

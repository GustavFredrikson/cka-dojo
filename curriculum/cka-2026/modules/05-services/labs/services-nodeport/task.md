# Expose an application on a fixed node port

In namespace `dojo-services-nodeport`, expose Deployment `web` as Service
`web-node` with these exact properties:

- type `NodePort`;
- Service port `80` targeting container port `80`;
- node port `30080`.

Do not modify the Deployment.

# Provide stable DNS discovery

In namespace `dojo-dns-service`, expose Deployment `api` using a ClusterIP
Service also named `api`. The Service must listen on port `8080`, target
container port `80`, and resolve by short DNS name from Deployment `client`.

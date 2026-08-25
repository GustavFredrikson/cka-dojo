# Read-only access to one namespace

A colleague, `alice`, needs to look at what is running in the `backend`
namespace so she can debug an application. She must not be able to change
anything, and she must not gain access to any other namespace.

Grant `alice`:

- read access (`get`, `list`, `watch`) to Pods in `backend`
- read access (`get`, `list`, `watch`) to Deployments in `backend`

She must **not** be able to:

- create, update or delete anything in `backend`
- read Secrets in `backend`
- read anything in any other namespace

There is no user object to create; `alice` is just a name the API server
authorizes against.

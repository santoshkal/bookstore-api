# Bookstore-api

A **WIP** DevOps project: a simple REST API built with Go, a frontend rendered with Go templates, and a Postgres database, all deployed to a Kubernetes Kind test cluster.

## Architecture

The application is split across two namespaces:

- **`frontend-api`** — the frontend and backend of the application together with all their dependencies.
- **`database`** — Postgres with its dependencies and environment variables.

## Accessing the service

The service can be reached in two ways:

- Via **NodePort**: `<node IP>:<service port>/books`
- Via **Ingress**: the `ingress` object is mapped to the host `app.santoshdts`. If you use this host, remember to map it in your `/etc/hosts` file.

Data is persisted even after node reboots by storing it on a `hostPath` volume.

## TODO

- [x] Apply GitOps functionality to the application for [CI with GitHub Actions and CD with Flux](https://santoshdts.hashnode.dev/a-step-by-step-guide-to-gitops-with-github-actions-and-flux2-including-a-hands-on-demo). 
- [x] Automate scanning and signing of build artefacts using syft and cosign. 
- [x] Create an Helm Chart for the application
- [x] Make the service available via `Kubernetes Ingress`
- [x] Maintains data persistancy after node reboots. Currently, I've used `hostPath` in Persistant Volumes mounted to StatefulSet, which is not reccomended practice to be used in production clusters.
- [x] Apply GitOps automation levereging Flux `OCIRepository` and `Kustomization` for storing all the relevant artifacts like images, deployment configs (kustomize and helm charts), cosign signatures, SBOM's, Kyverno/OPA policies,etc in an oci compliant registry (local docker registry in this case) and reconciling the cluster state with the `OCIRepository` and `Kustomization` Flux controllers.
    - [ ] A blog on hands demo about the subject published at Hashnode.
- [ ] Apply monitoring and observability.
    **Tools to work with**
    - Implement EFK stack
    - Instrument OpenTelemetry
    
- [ ] Remove static env Vars in Go application and make it receive the vars from the *configMap/secret*.
- [ ] Add more functionality and styling to the Go Apllication.


*If you happen to pass by this repo and would like to make a suggestion, add some functionality to it, or find something that is not handled appropriately, please feel free to file an issue or raise a PR. I would be happy to receive valuable input from you.

# Build a Cloud : Infrastructure → Platform → Product

This repository documents my journey through the **Build a Cloud** track, a hands-on cloud engineering program organized by **Arkadia** in collaboration with **STACKIT**.

The goal of this track is to build a complete cloud-native platform from the ground up starting with Infrastructure as a Service (IaaS), progressing through Kubernetes and Platform as a Service (PaaS), and ending with a production-ready platform featuring APIs, a web interface, automation, and observability.

Each week focuses on a different stage of the cloud platform lifecycle, with detailed implementation notes, architecture diagrams, and documentation available in the corresponding project folders and Notion pages.

---

## References

- [STACKIT](https://stackit.com/en)
- [Arkadia](https://arkadia.hn/)
- [LEVEL3](https://level3.hn/)
- [Build a Cloud Track](https://github.com/arkadiahn/LEVEL3-projects/tree/main/build-a-cloud)

---

# Week 1 : IaaS Foundations & Kubernetes on OpenStack

## Goal

Build a Kubernetes cluster on top of an OpenStack environment while learning the fundamentals of Infrastructure as a Service (IaaS) and Infrastructure as Code (IaC).

## What I Learned

- OpenStack architecture and core services
- Installing and configuring DevStack
- Validating OpenStack services
- Creating virtual machines
- Terraform basics (Providers, Resources, State)
- Provisioning infrastructure with Terraform
- Installing Kubernetes on an OpenStack VM
- Basic infrastructure automation
- Automating the entire setup with shell scripts
- Using Ansible to replace shell scripts for infrastructure provisioning and Kubernetes installation
- Implementing a one-command deployment that provisions an OpenStack VM and installs Kubernetes
- Expanding the automation to provision two virtual machines and configure a two-node Kubernetes cluster

## Documentation

The complete documentation, notes, screenshots, and implementation details are available on Notion:

**[🗒️ Week 1 Documentation](https://app.notion.com/p/Week-1-3a3faad93e9b801387d6e00fc224d82b?source=copy_link)**

# Week 2 : PaaS Product on STACKIT Kubernetes Engine (SKE)

## Goal

Build a managed Postgres PaaS product on top of STACKIT's managed Kubernetes offering (SKE), provisioned entirely as code, and deepen understanding of Kubernetes' extension mechanisms (CRDs, Operators, the Reconciler Pattern) along the way.

## What I Learned

- Provisioning a managed Kubernetes cluster (SKE) with Terraform, using `stackit_ske_cluster` and `stackit_ske_kubeconfig`
- Configuring a remote Terraform backend on STACKIT Object Storage (S3-compatible) for persistent, shareable state
- Debugging and working around a real upstream Terraform provider bug (`os_version_min` schema/state drift) with a safe, repeatable manual state-patch procedure
- Custom Resource Definitions (CRDs) and Custom Resources (CRs) , how Kubernetes is extended with new object types
- The Operator pattern and the Reconciler Loop: watch/informer mechanics, level-triggered reconciliation, idempotency, owner references, and finalizers
- Deploying CloudNativePG (CNPG), a production-grade Postgres Operator, via Helm
- Provisioning a real multi-instance, self-healing Postgres cluster (`Cluster` CR) with automatic primary/replica failover
- Kubernetes storage and networking primitives: PersistentVolumeClaims (PVCs), Services (`-rw`/`-ro`/`-r`), and how they're generated automatically by an Operator
- RBAC Roles, ClusterRoles, (Cluster)RoleBindings, and how a ServiceAccount's permissions gate what an Operator/controller can actually do
- GitOps principles, and the practical difference between imperative tools (`helm install`) and a continuously-reconciling, Git-sourced controller
- Installing and configuring ArgoCD, including Application sourcing directly from a Helm chart repo (not just a Git repo of raw manifests), sync-wave ordering, and automated sync/self-heal/prune
- CI/CD automation for infrastructure: a Forgejo Actions workflow that runs `terraform init/validate/plan/apply` automatically on every push to the infra folder

## Documentation

The complete documentation, notes, screenshots, and implementation details are available on Notion:

**[🗒️ Week 2 Documentation](https://app.notion.com/p/Week-2-3a9faad93e9b80ea97c7fedebe9ba2ad?source=copy_link)**

# Week 3 : Developing a Platform-as-a-Service (PaaS) Product: API Layer

## Goal

Expose the PaaS product through a clean, production-ready RESTful API to enable automated provisioning and seamless integration.

## What I Learned

- REST fundamentals , resources vs actions, HTTP method semantics (safe/idempotent vs. not), status code conventions, and statelessness
- Spec-first API design: writing the OpenAPI specification (`openapi.yaml`) before implementation
- Generating Go types, interfaces, and server boilerplate from an OpenAPI spec with `oapi-codegen`
- Building a Kubernetes-native API layer: using `client-go`'s dynamic client to create, list, get, and delete `Cluster` custom resources (`clusters.postgresql.cnpg.io`) directly from Go handlers
- Designing the product-instance lifecycle (create → list → get → connection info → delete) as a set of RESTful endpoints
- Adding authentication to the API: user registration/login backed by a database, with JWT issuance and verification middleware protecting all instance-management endpoints
- Structuring a Go project with generated code kept separate from hand-written business logic
- Containerizing a Go application with a multi-stage Dockerfile (built via `podman`, targeting `linux/amd64`)
- Publishing images to the STACKIT Container Registry and deploying via SKE, including RBAC via a dedicated ServiceAccount and image-pull authentication via a registry Secret
- Verifying in-cluster ServiceAccount-token authentication to the Kubernetes API from within a running pod

## Documentation

The complete documentation, notes, screenshots, and implementation details are available on Notion:

**[🗒️ Week 3 Documentation]()**


# Week 4 – Extending the Platform with Advanced Features

## Goal

Build a user-facing interface and expose the PaaS product securely via the web, ensuring a smooth and accessible developer experience.

## What I Learned

- Building a Vue.js web UI covering the platform's API functions (instance creation, listing, connection details, deletion)
- Implementing secure UI-to-backend communication using JWT
- Deploying and configuring Traefik as an Ingress controller on SKE, and how `type: LoadBalancer` triggers STACKIT's Cloud Controller Manager to provision a real external Network Load Balancer
- Layer 4 vs Layer 7 traffic handling: the NLB as a blind TCP relay vs Traefik reading HTTP and routing on Host/path via Ingress rules
- DNS zones and records: the difference between routing records (A/wildcard) and validation records (TXT), and why a zone is needed for both
- Provisioning a free STACKIT subdomain and DNS zone for the platform
- TLS certificate automation with cert-manager and the STACKIT DNS webhook, using the ACME DNS-01 challenge.
- The full TLS handshake and certificate trust chain, from ClientHello/SNI through to a Let's Encrypt–issued cert chaining to a browser-trusted root
- Why automated cert issuance needs its own least-privilege, dedicated Service Account, separate from broader infrastructure credentials
- Tracing a complete request end-to-end, DNS resolution → TCP handshake → TLS handshake → Ingress routing → Service → Pod → JWT verification → response
- Extending per-instance connectivity beyond the control-plane API: exposing individual database instances externally via dedicated per-instance LoadBalancer Services with IP-based access control (`loadBalancerSourceRanges`)

## Documentation

The complete documentation, notes, screenshots, and implementation details are available on Notion:

**[🗒️ Week 4 Documentation]()**
# INDIS — Infrastructure / Deployment

## Kubernetes

Kubernetes manifests and Helm charts for all INDIS services.

```
kubernetes/
├── base/               # Base manifests (Kustomize)
└── overlays/           # Environment-specific overlays
    ├── development/
    ├── staging/
    └── production/
```

## Terraform

Infrastructure provisioning for:
- Kubernetes cluster
- PostgreSQL (Identity DB)
- Redis cluster
- HSM integration
- Network security groups
- Monitoring stack

## Docker

Per-service Dockerfiles are co-located with each service.
This directory contains shared Docker configurations.

## Helm

Helm chart for deploying the complete INDIS stack.

## Requirements (PRD §6.1)

- **Kubernetes** for container orchestration
- **Helm** for deployment management
- **Terraform** for infrastructure as code
- **Istio** service mesh (mTLS, observability)
- All infrastructure on **sovereign** hardware — no foreign cloud providers

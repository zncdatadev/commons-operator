<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-04-06 | Updated: 2026-04-06 -->

# commons-operator

## Purpose
Provides shared utilities and common platform components for the Kubedoop operator ecosystem. Manages cross-cutting concerns including pod enrichment (injecting sidecars, volumes, env vars), restart policies, and shared CRD types (AuthenticationClass, S3Connection, S3Bucket) that are referenced by other operators.

## Key Files
| File | Description |
|------|-------------|
| `go.mod` | Go module: `github.com/zncdatadev/commons-operator` |
| `Makefile` | Build, generate, and deployment commands |
| `PROJECT` | Kubebuilder project metadata (domain: `kubedoop.dev`) |
| `Dockerfile` | Container image definition for the operator |
| `cmd/main.go` | Operator entry point; sets up manager and registers controllers |

## Subdirectories
| Directory | Purpose |
|-----------|---------|
| `api/` | CRD definitions: `authentication/v1alpha1`, `s3/v1alpha1` |
| `cmd/` | Operator entry point and manager setup |
| `config/` | Kubernetes manifests, RBAC, kustomize configs |
| `internal/controller/` | Reconciliation controllers: `pod_enrichment/`, `restart/` |
| `deploy/` | Helm chart for the operator |
| `test/` | E2E test suites using Ginkgo/Gomega |
| `hack/` | Code generation boilerplate |

## CRDs Managed
| Group | Kind | Description |
|-------|------|-------------|
| `authentication.kubedoop.dev` | `AuthenticationClass` | Authentication provider configuration |
| `s3.kubedoop.dev` | `S3Connection` | S3-compatible storage connection settings |
| `s3.kubedoop.dev` | `S3Bucket` | S3 bucket reference with connection binding |

## For AI Agents

### Working In This Directory
- Standard Kubebuilder operator structure (multigroup layout)
- Uses `operator-go` framework (`github.com/zncdatadev/operator-go v0.12.6`)
- Run `make generate` to regenerate CRD manifests and DeepCopy methods
- Run `make manifests` to update CRD YAML files in `config/crd/`
- Run `make test` for unit tests
- Run `make docker-build` to build the operator image
- Changes to shared CRDs or controllers here affect all dependent operators

### Testing Requirements
- Unit tests: `make test`
- E2E tests in `test/e2e/` — requires a running Kubernetes cluster
- E2E framework uses Ginkgo v2 + Gomega

### Common Patterns
- Controllers in `internal/controller/pod_enrichment/` and `internal/controller/restart/`
- All CRDs use `v1alpha1` API version
- Follows `operator-go` GenericReconciler pattern
- Multi-group API layout: `api/<group>/v1alpha1/`

## Dependencies

### Internal
- `../operator-go` — Shared operator framework (`github.com/zncdatadev/operator-go`)

### External
- `sigs.k8s.io/controller-runtime v0.23.0`
- `k8s.io/api`, `k8s.io/apimachinery`, `k8s.io/client-go v0.35.0`
- `github.com/onsi/ginkgo/v2`, `github.com/onsi/gomega` — E2E testing

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->

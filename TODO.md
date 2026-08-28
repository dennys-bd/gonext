*For Development agents*: when structuring something based on this doc be sure to think in complementary options, lets say that it states s3, so you would think on the development of it and suggest usage of minion or localstack images for s3. Also if creating some backend configuration you could check if there is any similar config for frontend or vice-versa;
Also, when finishing a Phase it can be removed from here.

## Phase 3: Security Baseline
- **Security Baseline**: Security headers middleware, CORS configuration, and rate limiting baseline.

## Phase 4: Frontend Scaffold & Full-Stack Integration
- **OpenAPI & Contract Sync**: OpenAPI schema static export and TypeScript client generation (e.g. `openapi-typescript` / `hey-api`).
- **Backend Integration**: Typed frontend API client, data fetching layer, and environment-based API base URL handling.
- **Auth Foundation**: Authentication middleware hook and user identity context injection into handlers.

## Phase 5: Containerization, Deployment & Scaffolding Tooling
- **Production Containerization**: Multi-stage production `Dockerfile`s for backend and frontend, update make target to use docker, user docker on CI.
- **Tools instalation by environment**: Maybe not using mise inside docker?
- **Infrastructure as Code (IaC) / Deployment**: Deployment manifests (Terraform/OpenTofu, Docker Compose prod, Fly.io, or K8s).
- **Scaffold Template Initializer**: Project initialization script (`init-project.sh`) to rename module paths, package names, and set up new repos from the template.


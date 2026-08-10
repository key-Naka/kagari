# ADR-0001: Separate Nuxt Web and Go API Services

- Status: Accepted
- Date: 2026-08-10

## Context

The site needs Nuxt 4 SSR pages and an owner-only administration console alongside a Go Fiber API for authentication, content management, visitor messages, media metadata, GitHub data, and server status. It will run on a cloud server already hosting another project behind Nginx and will be deployed through 1Panel with Docker.

## Decision

Run Nuxt 4 as the web service and Go Fiber as an independent API service. Route the public site and administration console through the main domain, route API traffic through a dedicated API subdomain, and keep MySQL, Redis, object storage credentials, and monitoring collection interfaces private to the deployment network.

## Consequences

- The frontend and backend can be deployed and scaled independently.
- API contracts must be explicit and versioned at the boundary.
- 1Panel/Nginx must terminate TLS and route the main and API hostnames to the correct containers.
- Local development requires two application processes plus infrastructure dependencies.

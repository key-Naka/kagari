# ADR-0002: Direct Go Collection for Public Service Status

- Status: Accepted
- Date: 2026-08-10

## Context

The site needs to show host, container, and application indicators. The owner chose not to introduce a separate Prometheus stack for the first architecture and wants the Go backend to collect the required data directly. The public status view must remain useful without exposing ports, process commands, environment variables, or other raw infrastructure identifiers.

## Decision

The Go service collects host metrics through platform APIs, reads permitted Docker container metrics through a restricted local interface, and performs application availability checks. It exposes only an aggregated, sanitized Service Status response; the Administration Console does not provide a separate raw-metrics view in the first release.

## Consequences

- The deployment has fewer monitoring dependencies.
- The Go service owns collection scheduling, timeout handling, caching, and metric normalization.
- The collector must run with least-privilege access to Docker and host metrics.
- A future Prometheus migration remains possible if metric cardinality or retention requirements grow.

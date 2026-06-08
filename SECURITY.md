# Security Policy

## Secret Handling

This repository is prepared for public delivery. Do not commit:

- `.env` files with real values
- server usernames, passwords, or SSH private keys
- JWT secrets
- MySQL, Redis, PostgreSQL, or Grafana passwords
- TOS, DashScope, OpenAI, or other API keys
- private IP login records or deployment notes containing credentials

Use `.env.example` files for configuration names and placeholders only.

## Reporting Issues

If you find a leaked credential or sensitive artifact, rotate the affected secret first, then remove it from the repository and history before making the repository public.

## Public Delivery Boundary

The public main branch represents the challenge delivery implementation. Experimental distributed-cluster work is intentionally excluded from this branch unless it is separately reviewed, sanitized, and documented.

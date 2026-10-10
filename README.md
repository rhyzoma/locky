# 🐹🔐 Locky

> A small, friendly authentication service for modern microservice architectures.

Locky is a reusable authentication service for applications built from independent services.
It handles the parts of authentication that should be boring and dependable: accounts,
credentials, email verification, sessions, refresh tokens, and access-token issuance.

It can run behind an API gateway, alongside a backend-for-frontend, or as a standalone
internal service consumed directly by trusted applications and services.

Profile data belongs to `profile-service`. Groups, roles, permissions, and ACL belong to an authorization service.

## What Locky does

- Register and authenticate users.
- Hash passwords with bcrypt.
- Issue short-lived access tokens and rotating refresh tokens.
- Verify email addresses with single-use, expiring tokens.
- Revoke one session or all sessions for a user.
- Expose a gRPC API for the API gateway and other trusted services.
- Persist production data in PostgreSQL through sqlc-generated queries.

## Architecture

```text
                    ┌────────────────────┐
                    │       Locky        │
                    │ authentication     │
                    │                    │
                    │ user + credential  │
                    │ session + tokens   │
                    └─────────────┬──────┘
                                  │ JWT / user_id
                                  ▼
                              API gateway
                              │        │
                              ▼        ▼
                       profile-service  business services
```

Locky owns accounts and authentication state, not every piece of user information. Other services reference the stable
`user_id` issued by Locky. The API gateway owns orchestration with profile-service and other
business services; Locky is responsible only for accounts, credentials, sessions, and tokens.

## Quick start

Requirements:

- Docker
- PostgreSQL and SMTP for production mode

Available configurations:

| File | Usage |
| --- | --- |
| [`configs/locky.local.yaml`](configs/locky.local.yaml) | Local development with in-memory storage and automatic email verification. |
| [`configs/locky.production.yaml`](configs/locky.production.yaml) | Production baseline; requires database, TLS, JWT, and SMTP environment values. |

Build the image from this directory:

```powershell
docker build -t locky:latest .
```

Run with the local configuration:

```powershell
docker run --rm --name locky `
  -p 8080:8080 `
  -e CONFIG_FILE=/etc/locky/config.yaml `
  -v "${PWD}/configs/locky.local.yaml:/etc/locky/config.yaml:ro" `
  locky:latest
```

Run with the production configuration:

```powershell
docker run --rm --name locky `
  -p 8080:8080 `
  -e CONFIG_FILE=/etc/locky/config.yaml `
  -e DATABASE_URL="postgres://user:password@host.docker.internal:5432/locky?sslmode=require" `
  -e JWT_SIGNING_KEY="a-random-secret-at-least-32-characters-long" `
  -e JWT_KEY_ID="2026-01" `
  -e TLS_CERT_FILE="/run/secrets/locky.crt" `
  -e TLS_KEY_FILE="/run/secrets/locky.key" `
  -e SMTP_HOST="smtp.example.com" `
  -e SMTP_PORT="587" `
  -e SMTP_FROM="noreply@example.com" `
  -e VERIFY_URL="https://app.example.com/verify-email" `
  -v "${PWD}/configs/locky.production.yaml:/etc/locky/config.yaml:ro" `
  -v "${PWD}/certs:/run/secrets:ro" `
  locky:latest
```

The image reads `CONFIG_FILE` when the container starts. Config files and secrets stay outside the image, and the container runs as a non-root user. Apply database migrations before starting the production container.

## Security notes

- Never use `configs/locky.local.yaml` in production.
- Keep JWT secrets in a secret manager or protected environment configuration.
- Set `JWT_KEY_ID` to identify the active signing key. The value is emitted as
  the JWT `kid` header so consumers can perform coordinated key rotation.
- Keep access tokens short-lived and rotate refresh tokens.
- Use TLS for service-to-service traffic.
- Put distributed rate limiting at the gateway or use a shared limiter when running multiple Locky replicas.

## JWT contract

Locky issues HMAC-SHA256 access tokens with these stable claims:

- `sub`: Locky user ID;
- `email`: normalized email address;
- `iss`: `locky`;
- `aud`: `locky`;
- `iat` and `exp`: issue and expiry timestamps;
- `kid`: configured signing-key identifier.

The API gateway or another trusted consumer is responsible for validating the
signature, issuer, audience, expiry, and expected algorithm. During key
rotation, deploy consumers with the new key while retaining the old key for
the maximum access-token lifetime, switch `JWT_SIGNING_KEY` and
`JWT_KEY_ID`, then remove the old key after existing tokens expire. HMAC keys
must be distributed through a secret manager; Locky does not expose them.

## Operations

Locky registers the standard gRPC health service. Probe the service name
`locky.v1.LockyService`; it reports `SERVING` only after the store and server
are initialized and changes to `NOT_SERVING` during shutdown. A deployment
should use the health endpoint for readiness and remove the instance from
traffic before termination.

The gRPC server also recovers handler panics and returns an internal error
instead of terminating the process. Application logs use structured fields so
they can be collected by the deployment platform.

## License

Locky is distributed under the [MIT License](LICENSE).

# Security Policy

## Supported versions

Security fixes are applied to the latest release on the default branch. Use a
released version rather than an arbitrary commit in production.

## Reporting a vulnerability

Do not open a public issue for a suspected vulnerability. Contact the
maintainer privately through the repository owner's security contact and
include:

- affected version or commit;
- deployment and configuration details;
- reproduction steps or a proof of concept;
- the impact you observed.

Please allow reasonable time for triage and a coordinated fix before public
disclosure.

## Deployment requirements

- Set a unique JWT signing key of at least 32 characters in production.
- Keep the key in a secret manager and never commit it to a config file.
- Use TLS for gRPC traffic.
- Put a distributed rate limiter at the API gateway when running replicas.
- Apply database migrations before starting a release.

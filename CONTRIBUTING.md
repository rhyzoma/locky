# Contributing to Locky

## Development checks

Before opening a pull request, run:

```bash
gofmt -w .
go mod verify
go vet ./...
go test ./...
go test -race -short ./...
golangci-lint run ./...
```

Keep generated protobuf and sqlc files in sync with their source files. Do not
edit generated files by hand.

## Pull requests

- Explain the behaviour change and its compatibility impact.
- Add or update tests for changed behaviour.
- Update the README or security documentation when configuration or API
  contracts change.
- Keep authentication, profile, and authorization responsibilities separate.

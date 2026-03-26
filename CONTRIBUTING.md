# Contributing to Neocoin

## Development Status

Neocoin is a **production-ready PoW blockchain**. Core consensus is stable. Contributions related to:

- ✅ Bug fixes
- ✅ Test coverage
- ✅ Documentation
- ✅ Protocol spec improvements
- ⚠️ New features (please open an issue first to discuss)

## Prerequisites

- Go 1.21+
- Docker & Docker Compose
- `gofmt`, `go vet`

## Running Tests

```bash
cd blockchain
go test ./...
go vet ./...
gofmt -d .
```

## Running the Smoke Test

```bash
cd ..
./scripts/smoke_test.sh
```

## Code Style

- Run `gofmt -w .` before submitting
- Add tests for new functionality
- Keep PRs small and focused

## Submitting Changes

1. Fork the repo (if external)
2. Create a feature branch
3. Make changes + add tests
4. Run `go test ./...` and `go vet ./...`
5. Submit a pull request

## Reporting Bugs

Please include:
- Steps to reproduce
- Expected vs actual behavior
- Go version, Docker version
- Relevant logs

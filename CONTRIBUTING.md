# Contributing to libdns/bluecat

Thank you for contributing to the Bluecat provider for libdns!

## Development Setup

1. Clone the repository:
```bash
git clone https://github.com/libdns/bluecat.git
cd bluecat
```

2. Install dependencies:
```bash
go mod tidy
```

3. Build the package:
```bash
go build ./...
```

## Testing

### Unit Tests

Run the unit tests:
```bash
go test -v ./...
```

### Integration Tests

To run integration tests against a live Bluecat instance, set the following environment variables:

```bash
export BLUECAT_SERVER_URL="https://your-bluecat-server.com"
export BLUECAT_USERNAME="your-username"
export BLUECAT_PASSWORD="your-password"
export BLUECAT_TEST_ZONE="yourtestzone.com."
```

Then run:
```bash
go test -v ./...
```

**Important:** Never commit credentials to the repository. The `.gitignore` file is configured to exclude credential files.

## Code Style

- Follow standard Go formatting with `gofmt`
- Use `go vet` to check for common mistakes
- Add godoc comments for all exported types and functions
- Keep functions focused and well-documented

## Pull Request Process

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/your-feature`)
3. Make your changes
4. Run tests to ensure everything works
5. Commit your changes with clear commit messages
6. Push to your fork and submit a pull request

## Requirements for Contributions

As per the [libdns implementation guidelines](https://github.com/libdns/libdns/wiki/Implementing-a-libdns-package):

- All exported fields, methods, and functions must have godoc comments
- Use `json` struct tags with `"snake_case,omitempty"` convention
- Run `go mod tidy` before committing
- Minimize dependencies
- Must be pure Go (no cgo)
- Configuration primarily through struct fields
- All methods must be thread-safe
- Adhere to libdns interface semantics

## Reporting Issues

If you find a bug or have a feature request, please open an issue on GitHub with:
- A clear description of the problem
- Steps to reproduce (if applicable)
- Expected vs actual behavior
- Go version and OS information

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

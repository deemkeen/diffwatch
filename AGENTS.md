# Agent Guidelines

## Build & Test Commands
- Build: `go build -o bin/diffwatch ./cmd/diffwatch`
- Run: `go run ./cmd/diffwatch -path <directory>`
- Lint: `golangci-lint run` (if installed) or `go vet ./...`
- Format: `go fmt ./...` or `gofmt -s -w .`
- Test all: `go test ./...`
- Test single: `go test ./internal/<package> -run <TestName>`
- Type check: Go compiler handles this during build

## Code Style
- **Imports**: Group by external dependencies, internal modules, then relative imports. Sort alphabetically within groups.
- **Formatting**: Follow project formatter configuration (check for .editorconfig, .prettierrc, or similar).
- **Types**: Use strict typing. Define explicit types/interfaces - avoid dynamic/any types.
- **Naming**: Follow language conventions - check existing code for patterns (camelCase, snake_case, PascalCase, etc.).
- **Functions**: Keep functions small and focused. One responsibility per function.
- **Error Handling**: Always handle errors explicitly. Use language-appropriate error handling patterns.
- **Comments**: Document public APIs. Use inline comments sparingly for complex logic only.
- **Files**: Organize by feature or domain. Follow existing project structure patterns.

## Project Conventions
- **Before coding**: Read similar files to understand existing patterns and conventions.
- **Before committing**: Run formatter, linter, and relevant tests.
- **Architecture**: Maintain separation of concerns. Don't mix business logic with I/O or presentation.
- **Testing**: Write tests for new features. Update tests when modifying existing code.
- **Dependencies**: Check if functionality exists in project before adding new dependencies.

Thank you for your interest in contributing to TabSplit Backend!

This document explains how to contribute code, run tests, and prepare a clean pull request.

1. How to contribute

- Fork the repository.
- Create a descriptive branch from `main`: `git checkout -b feat/short-description` or `fix/bug-brief`.
- Make your changes. Keep commits small and focused.
- Run tests and linters locally (see below).
- Push your branch and open a Pull Request against `main` with a clear summary and description of changes.

2. Development workflow

- Keep changes isolated to a single logical task per PR.
- Include tests for new features or bug fixes when applicable.
- Mention related issues in the PR (e.g., `Fixes #123`).

3. Code style and checks

- Format Go code with `gofmt`:

```powershell
gofmt -w .
```

- Vet your code:

```powershell
go vet ./...
```

- Run unit tests:

```powershell
go test ./...
```

- Prefer using idiomatic Go code and small functions. Keep packages focused.

4. Commit message style

- Use short, descriptive commit messages. Example:

```
feat(auth): add JWT support for login
fix(db): handle nil pointer when user not found
```

5. Branch naming

- Use `feat/`, `fix/`, `chore/`, `docs/`, `test/` prefixes, followed by a short dash-separated description.

6. Pull Request checklist

- [ ] The code builds and tests pass locally
- [ ] Code is formatted with `gofmt`
- [ ] New behavior is covered by tests (where applicable)
- [ ] PR description explains the motivation and changes

7. Running the project locally

- Ensure Go 1.18+ is installed.
- Set database environment variables required by `src/Database/connection.go`.
- Start the app:

```powershell
# from repository root
$env:DB_HOST = 'localhost'; $env:DB_PORT = '5432'; $env:DB_USER = 'postgres'; $env:DB_PASSWORD = 'yourpassword'; $env:DB_NAME = 'tabsplit'
go run .\src\main.go
```

8. CI and automated checks

- Pull Requests will run formatting, vet, and tests (if CI is configured). Please fix any issues reported by CI before requesting review.

9. Security & sensitive data

- Do not commit secrets, credentials, or private keys. Use environment variables or a secrets manager.
- If you discover a security vulnerability, open an issue and mark it private or contact the repository owner to coordinate disclosure.

10. Questions or help

- If you're unsure where to start, open an issue describing what you want to do. Maintainers will help point you to a good starter task.

Thanks for contributing — your help makes the project better!

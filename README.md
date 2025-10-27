# TabSplit Backend

A small, idiomatic Go backend for TabSplit — split group expenses and keep track of users and payments.

## What this repo contains

- `src/` — application source code
  - `main.go` — application entry point
  - `Database/connection.go` — database connection helper
  - `models/` — data models (e.g. `User.go`)

## Prerequisites

- Go 1.18+ installed (Go 1.20+ recommended). Verify with:

```powershell
go version
```

- Git (to clone and manage contributors)

Optional:

- A local PostgreSQL / MySQL / SQLite instance depending on how `src/Database/connection.go` is configured. Check that file for the exact driver and connection string format.

## Quick start (development)

1. Clone the repo:

```powershell
git clone https://github.com/Tjens23/tabsplit-backend.git
cd tabsplit-backend
```

2. Fetch dependencies:

```powershell
go mod download
```

3. Configure environment variables

Create a `.env` file or set environment variables used by `src/Database/connection.go`. Typical variables to set (adjust names to match your code):

```
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=yourpassword
DB_NAME=tabsplit
```

On Windows PowerShell you can set them for the current session like this:

```powershell
$env:DB_HOST = 'localhost'
$env:DB_PORT = '5432'
$env:DB_USER = 'postgres'
$env:DB_PASSWORD = 'yourpassword'
$env:DB_NAME = 'tabsplit'
```

4. Run the app (development):

```powershell
go run .\src\main.go
```

Or build a binary and run it:

```powershell
go build -o bin\tabsplit .\src
.\bin\tabsplit
```

## Tests

If there are tests, run them with:

```powershell
go test ./...
```

## Database notes

The DB connection logic lives in `src/Database/connection.go`. Open that file to confirm:

- database driver (Postgres/MySQL/SQLite)
- expected DSN / environment variable names

If migrations are used, add migration steps here (e.g., with `golang-migrate` or an ORM-specific tool).

## Project structure

Keep the project structure tidy. A suggested structure (already mostly used):

- `src/` - app source
  - `main.go` - entry
  - `Database/` - connection + migration helpers
  - `models/` - domain models

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feat/your-feature`
3. Make changes and add tests where appropriate
4. Run `go test ./...` and ensure the build passes
5. Open a Pull Request against `main` with a clear description

Follow standard Go formatting and vetting:

```powershell
gofmt -w .
go vet ./...
```

## Contributors

- Tjens23 (repository owner)

To generate an up-to-date contributors list locally run:

```powershell
git shortlog -sn --no-merges
```

This will show commit counts and contributor names based on your git history.

## License

This project does not contain a LICENSE file yet. Consider adding an open-source license (for example, MIT) in a `LICENSE` file.

## Troubleshooting

- If `go run` fails with module errors, run `go mod tidy` then `go mod download`.
- If the app can't connect to the DB, check the environment variables and ensure the DB service is running and reachable.

## Next steps / extras

- Add a `Makefile` or PowerShell script for common tasks (`build`, `run`, `migrate`).
- Add CI (GitHub Actions) to run `gofmt`, `go vet`, `go test` on PRs.
- Add a `LICENSE` file and a `CONTRIBUTING.md` when the project grows.

---

If you'd like, I can also:

- Add a `LICENSE` file (MIT) and a `CONTRIBUTING.md` template
- Create a tiny PowerShell script for local startup
- Add basic GitHub Actions workflow to run tests on PRs

Let me know which extras you want and I can add them.

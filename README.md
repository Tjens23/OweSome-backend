# OweSome Backend

- `src/` — application source code
  - `main.go` — application entry point with Fiber v3 web server
  - `Controllers/` — API endpoint handlers (Auth, User, Group, Expense)
  - `Database/` — database connection and models
    - `connection.go` — PostgreSQL connection with GORM
    - `models/` — data models (User, Group, GroupMember, Expense, ExpenseShare, Settlement, RefreshToken)
  - `middleware/` — authentication middleware
  - `Routes/` — API route definitions
  - `docs/` — auto-generated Swagger documentation
  - `static/` — static files (Swagger UI)

## Features

- 🔐 **JWT Authentication** with refresh tokens (24h access, 7d refresh)
- 👥 **User Management** with secure password hashing
- 🏠 **Group Management** with admin controls
- 💰 **Expense Tracking** with automatic split calculations
- 📊 **Swagger Documentation** with interactive UI
- 🔄 **Token Refresh System** for seamless authentication
- 🛡️ **Secure Middleware** for protected routes

## Prerequisites

- Go 1.20+ installed. Verify with:

```powershell
go version
```

- PostgreSQL database instance
- Git (to clone and manage contributors)

## Quick start (development)

1. Clone the repo:

```powershell
git clone https://github.com/Tjens23/OweSome-backend.git
cd OweSome-backend
```

2. Fetch dependencies:

```powershell
go mod download
```

3. Configure environment variables

Create a `.env` file in the root directory:

```env
DATABASE_URL=postgres://username:password@localhost:5432/owesome?sslmode=disable
```

Or set environment variables for the current PowerShell session:

```powershell
$env:DATABASE_URL = 'postgres://username:password@localhost:5432/owesome?sslmode=disable'
```

4. Run the application:

```powershell
go run .\src\main.go
```

The server will start on `http://localhost:3001`

## API Documentation

Once the server is running, access the interactive Swagger documentation:

- **Swagger UI**: http://localhost:3001/swagger
- **Raw JSON**: http://localhost:3001/swagger/doc.json

## API Endpoints

### Authentication

- `POST /auth/login` - User login (returns access + refresh tokens)
- `POST /auth/logout` - User logout (revokes all tokens)
- `POST /auth/refresh` - Refresh access token
- `GET /auth/user` - Get current user info

### Users

- `GET /users` - Get all users
- `POST /users` - Create new user
- `PATCH /users/update/:id` - Update user
- `DELETE /users/delete/:id` - Delete user

### Groups

- `GET /groups` - Get user's groups
- `POST /groups` - Create new group
- `PATCH /groups/update/:id` - Update group (admin only)
- `DELETE /groups/delete/:id` - Delete group (admin only)

### Expenses

- `GET /expenses` - Get expenses
- `POST /expenses` - Create expense
- `GET /expenses/:id` - Get expense details
- `PATCH /expenses/update/:id` - Update expense
- `DELETE /expenses/delete/:id` - Delete expense

## Database Schema

The application uses PostgreSQL with GORM for ORM. Database tables are auto-migrated on startup:

- **users** - User accounts with hashed passwords
- **refresh_tokens** - JWT refresh tokens with expiration tracking
- **groups** - Expense groups with admin management
- **group_members** - User membership in groups
- **expenses** - Shared expenses with amounts and descriptions
- **expense_shares** - Individual user shares of expenses
- **settlements** - Payment settlements between users

## Authentication System

### JWT Token System

- **Access Token**: 24-hour expiration, used for API authentication
- **Refresh Token**: 7-day expiration, used to generate new access tokens
- **Security**: Tokens stored as HTTP-only cookies + returned in response body

### Token Refresh Flow

1. Login → Receive access token (24h) + refresh token (7d)
2. Access token expires → Use refresh token to get new tokens
3. Logout → All tokens revoked from database

## Development

### Generate Swagger Documentation

```powershell
# Install swag CLI tool
go install github.com/swaggo/swag/cmd/swag@latest

# Generate docs (run from src/ directory)
cd src
swag init
```

### Build Application

```powershell
# Development
go run .\src\main.go

# Production build
go build -o bin\owesome.exe .\src\main.go
.\bin\owesome.exe
```

## Project Structure

```
OweSome-backend/
├── src/
│   ├── main.go                 # Application entry point
│   ├── Controllers/            # API endpoint handlers
│   │   ├── AuthController.go   # Authentication & token management
│   │   ├── UserController.go   # User CRUD operations
│   │   ├── GroupController.go  # Group management
│   │   └── ExpenseController.go # Expense tracking
│   ├── Database/
│   │   ├── connection.go       # PostgreSQL connection
│   │   └── models/             # Database models
│   │       ├── User.go
│   │       ├── Group.go
│   │       ├── GroupMember.go
│   │       ├── Expense.go
│   │       └── RefreshToken.go
│   ├── middleware/
│   │   └── isAuth.go          # JWT authentication middleware
│   ├── Routes/
│   │   └── Routes.go          # API route definitions
│   ├── docs/                  # Auto-generated Swagger docs
│   │   ├── docs.go
│   │   ├── swagger.json
│   │   └── swagger.yaml
│   └── static/
│       └── swagger-ui.html    # Swagger UI interface
├── go.mod                     # Go module dependencies
├── go.sum                     # Dependency checksums
├── .env                       # Environment variables
└── README.md                  # This file
```

## Technology Stack

- **Backend**: Go 1.20+ with Fiber v3 web framework
- **Database**: PostgreSQL with GORM ORM
- **Authentication**: JWT with refresh token rotation
- **Documentation**: Swagger/OpenAPI 3.0
- **Security**: bcrypt password hashing, HTTP-only cookies

## Environment Variables

Create a `.env` file with the following variables:

```env
# Database Configuration
DATABASE_URL=postgres://username:password@localhost:5432/owesome?sslmode=disable

# Optional: JWT Secret (defaults to "supersecretstring")
JWT_SECRET=your-super-secure-secret-key

# Optional: Server Port (defaults to 3001)
PORT=3001
```

## Example Usage

### User Registration & Login

```bash
# Create a user
curl -X POST http://localhost:3001/users \
  -H "Content-Type: application/json" \
  -d '{"username": "john", "password": "password123", "email": "john@example.com"}'

# Login
curl -X POST http://localhost:3001/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "john", "password": "password123"}'
```

### Create a Group & Add Expenses

```bash
# Create a group (requires authentication)
curl -X POST http://localhost:3001/groups \
  -H "Content-Type: application/json" \
  -H "authorization:  bearer <token> \
  -d '{"name": "Vacation Trip", "description": "Beach vacation expenses"}'

# Create an expense in the group
curl -X POST http://localhost:3001/expenses \
  -H "Content-Type: application/json" \
  -H "authorization:  bearer <token> \
  -d '{"amount": 120.50, "description": "Dinner at restaurant", "group_id": <id>}'
```

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/your-feature`
3. Make changes and add Swagger documentation for new endpoints
4. Test your changes locally
5. Run code formatting: `gofmt -w .`
6. Run linting: `go vet ./...`
7. Open a Pull Request with a clear description

## Contributors

- **Tjens23** - Project Owner & Lead Developer

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Troubleshooting

### Common Issues

**Module errors**

```powershell
go mod tidy
go mod download
```

**Database connection issues**

- Verify PostgreSQL is running: `pg_ctl status`
- Check DATABASE_URL format in `.env` file
- Ensure database exists: `createdb owesome`

**Port already in use**

- Change PORT in `.env` file
- Kill existing process: `netstat -ano | findstr :3001`

**Swagger documentation not updating**

```powershell
cd src
swag init
```

**JWT token issues**

- Clear browser cookies for localhost:3001
- Check token expiration (access: 24h, refresh: 7d)
- Use `/auth/refresh` endpoint for new tokens

---

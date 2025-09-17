# Go Auth Boilerplate Project

A RESTful API built with Go, Fiber, PostgreSQL, and Redis for managing users and their posts.

## Features

- User authentication (signup, login, logout)
- Session management with Redis
- CRUD operations for posts
- PostgreSQL database with GORM
- Swagger documentation
- Request validation
- Pagination for posts listing

## Prerequisites

- Go 1.21 or higher
- PostgreSQL
- Redis
- golang-migrate (for database migrations)

## Setup

1. Clone the repository:
```bash
git clone https://github.com/yourusername/go-auth-boilerplate.git
cd go-auth-boilerplate
```

2. Install dependencies:
```bash
go mod download
```

3. Create a `.env` file in the config directory with the following content based on example inside config folder

4. Create the database:
```bash
docker compose --env-file config/development.env down -v && docker compose --env-file config/development.env up -d
```

5. Run migrations: <br />
For migration tool run `brew install golang-migrate`. Before run the command, keep in mind that the port for DB from your env. In the case of development.env - `5433`
```bash
migrate -path migrations -database "postgresql://postgres:postgres@localhost:5433/go_auth_boilerplate?sslmode=disable" up
```

6. Run seeds:
```bash
go run cmd/seed/main.go
```

## Running the Application

1. Start the server:
```bash
go run main.go
```

2. The server will start at `http://localhost:9999`

## API Documentation

Swagger documentation is available at `http://localhost:9999/swagger/`

## API Endpoints

### Authentication
- `POST /api/v1/user/signup` - Create a new user account
- `POST /api/v1/user/login` - Login with email and password
- `POST /api/v1/user/logout` - Logout current user
- `PATCH /api/v1/user/update_password` - Update user password

### User
- `GET /api/v1/session` - Get current user information

### Posts
- `POST /api/v1/posts/create` - Create a new post
- `GET /api/v1/posts` - Get all posts (paginated)
- `GET /api/v1/posts/:id` - Get a specific post
- `PATCH /api/v1/posts/:id/update` - Update a post
- `DELETE /api/v1/posts/:id/delete` - Delete a post

## Test User

The application comes with a test user:
- Email: antonkalik@gmail.com
- Password: Pass123

This user is automatically created when running in development mode and comes with 20 sample posts.

## Development

To run the application in development mode with seeded data:
```bash
GO_ENV=development go run main.go
``` 
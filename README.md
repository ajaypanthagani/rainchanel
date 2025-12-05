# RainChanel

A distributed task queue system for executing WebAssembly (WASM) modules.

## Features

- User authentication with JWT
- Task publishing and consumption
- WASM module validation and execution
- Result publishing and consumption
- MySQL database persistence

## Prerequisites

- Go 1.24.3+
- MySQL 5.7+

## Setup

1. Clone the repository:
```bash
git clone <repository-url>
cd rainchanel
```

2. Install dependencies:
```bash
go mod download
```

3. Configure the database in `application.yaml`:
```yaml
database:
  host: 127.0.0.1
  port: 3306
  user: root
  password: ""
  database: rainchanel
```

4. Run the application:
```bash
go run cmd/main.go
```

The server will start on port 8080 by default.

## API Endpoints

### Public Endpoints

- `POST /register` - Register a new user
- `POST /login` - Login and get JWT token

### Protected Endpoints (require JWT token in Authorization header)

- `POST /tasks` - Publish a task
- `GET /tasks` - Consume a task
- `POST /results` - Publish a result
- `GET /results` - Consume a result

## License

MIT


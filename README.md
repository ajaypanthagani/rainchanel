# RainChanel

A distributed task queue system for executing WebAssembly (WASM) modules.

## Features

- User authentication with JWT
- Task publishing and consumption
- WASM module validation (execution handled by workers)
- Result publishing and consumption
- MySQL database persistence
- **Automatic task retries** with exponential backoff
- **Stale task detection** and automatic reclaim
- **Structured logging** (JSON or human-readable)
- **Prometheus metrics** endpoint
- **Health check** endpoint with queue statistics
- **Database indexes** for optimal performance

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

3. Configure the database and task settings in `application.yaml`:
```yaml
database:
  host: 127.0.0.1
  port: 3306
  user: root
  password: ""
  database: rainchanel

task:
  timeout_seconds: 300  # Max execution time for a task (5 minutes)
  max_retries: 3        # Maximum number of retry attempts
  stale_check_interval_seconds: 30  # How often to check for stale tasks
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
- `GET /health` - Health check endpoint with queue statistics
- `GET /metrics` - Prometheus metrics endpoint
- `GET /ping` - Simple ping endpoint

### Protected Endpoints (require JWT token in Authorization header)

- `POST /tasks` - Publish a task
- `GET /tasks` - Consume a task (returns oldest pending task)
- `POST /results` - Publish a successful result
- `POST /failures` - Publish a task failure (triggers automatic retry if retries available)
- `GET /results` - Consume a result for the authenticated user

## Task Lifecycle

1. **Publish Task**: Client publishes a task with WASM module, function name, and arguments
2. **Consume Task**: Worker polls `GET /tasks` to claim a pending task
3. **Execute**: Worker executes the WASM module (outside this system)
4. **Publish Result/Failure**: 
   - Worker calls `POST /results` on success
   - Worker calls `POST /failures` on failure (triggers automatic retry with exponential backoff)
5. **Consume Result**: Task creator polls `GET /results` to get their results

## Automatic Features

- **Stale Task Detection**: Background service automatically detects tasks that have been processing longer than the timeout and reclaims them
- **Automatic Retries**: Failed tasks are automatically retried up to `max_retries` times with exponential backoff
- **Task Timeout**: Tasks that exceed `timeout_seconds` are automatically reclaimed or marked as failed

## Configuration

All configuration can be set via `application.yaml` or environment variables:

- `TASK_TIMEOUT_SECONDS` - Max execution time for a task
- `TASK_MAX_RETRIES` - Maximum number of retry attempts
- `STALE_CHECK_INTERVAL_SECONDS` - How often to check for stale tasks
- `LOG_FORMAT` - Set to `json` for structured JSON logging

## Observability

- **Structured Logging**: Set `LOG_FORMAT=json` for JSON logs with task_id, user_id, etc.
- **Metrics**: Prometheus-compatible metrics at `/metrics` endpoint
- **Health Check**: Queue statistics and database health at `/health` endpoint

## License

MIT


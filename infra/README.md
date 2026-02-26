# RealChat Infrastructure

This directory contains the infrastructure configuration for local development.

## Structure

- `docker-compose.yml`: Main orchestration file.
- `.env`: Environment variables (credentials, endpoints).
- `config/`: Service-specific configurations (Prometheus, Grafana, Nginx, Postgres init).
- `data/`: Persistent runtime data (Postgres files, etc. Git ignored).

## Usage

Use standard `docker compose` commands from this directory:

### Start Infrastructure
```powershell
# From the infra/ directory
docker compose up -d
```

### Build and Start a Specific Service
```powershell
docker compose up -d --build <service-name>
```

### Check Logs
```powershell
docker compose logs -f
```

### Stop Everything
```powershell
docker compose down
```

### Clean State (Remove volumes)
```powershell
docker compose down -v
```

## Services

| Service | Port | Description |
|---------|------|-------------|
| API Gateway | 8080 | Entry point for REST/gRPC |
| Nginx | 80 | Entry point for WebSockets |
| Grafana | 3000 | Metrics visualization |
| Prometheus | 9090 | Metrics collection |
| Jaeger | 16686| Tracing UI |
| Postgres | 5432 | Primary database |
| Redis | 6379 | Caching and Presence |
| Kafka | 9092 | Event streaming |

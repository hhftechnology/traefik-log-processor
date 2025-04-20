# Traefik Log Processor

A lightweight, resource-efficient tool that splits Traefik logs by service name while maintaining the original JSON format.

## Features

- Splits Traefik JSON logs based on `ServiceName` field
- Preserves original log format and structure
- Supports multiple input methods (file, directory monitoring, stdin)
- Configurable log rotation (size-based and time-based)
- Configurable log retention policies (age-based and count-based)
- Minimal resource footprint (written in Go)
- Runs in a lightweight container
- Simple configuration via YAML file

## Quick Start

```bash
# Using Docker
docker run -v /path/to/traefik/logs:/logs -v /path/to/output:/output \
  -v /path/to/config.yaml:/app/config.yaml \
  ghcr.io/hhftechnology/traefik-log-processor:latest

# Using Docker Compose
docker compose up -d
```

## Configuration

Create a `config.yaml` file:

```yaml
input:
  # Watch a single file
  file: "/logs/traefik.log"
  
  # Or watch a directory for log files
  # directory: "/logs"
  # pattern: "*.log"
  
  # Or read from stdin
  # stdin: true

output:
  # Base directory for service-specific logs
  directory: "/output"
  
  # Format for service directories (supports templating)
  format: "{{.ServiceName}}"

rotation:
  # Maximum size of each log file in MB
  max_size: 100
  
  # Maximum age of each log file in days
  max_age: 7
  
  # Maximum number of old log files to retain
  max_backups: 5
  
  # Whether to compress old log files
  compress: true

# Optional field mapping (useful for CLF format or adding fields)
field_mapping:
  # You can rename fields or add new computed fields
  # Example for adding a RouterName field:
  # router_name: "{{extractRouterName .ServiceName}}"
```

## How It Works

1. The application watches Traefik log files or reads from stdin
2. Each log line is parsed as JSON
3. The `ServiceName` field is extracted (e.g., `5-service@http`)
4. The log entry is written to a service-specific directory
5. Log rotation and retention policies are applied to manage storage

## Building from Source

```bash
git clone https://github.com/hhftechnology/traefik-log-processor.git
cd traefik-log-processor
go build -o traefik-log-processor cmd/main.go
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
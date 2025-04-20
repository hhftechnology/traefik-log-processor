# Traefik Log Splitter Usage Examples

This document provides examples of how to use the Traefik Log Splitter in various scenarios.

## Basic Usage

### Using Docker

```bash
docker run -v /path/to/traefik/logs:/logs:ro -v /path/to/output:/output \
  ghcr.io/hhftechnology/traefik-log-processor:latest
```

### Using Docker Compose

1. Create a `docker-compose.yml` file:

```yaml
services:
  traefik-log-processor:
    image: ghcr.io/hhftechnology/traefik-log-processor:latest
    volumes:
      - /path/to/traefik/logs:/logs:ro
      - /path/to/output:/output
```

2. Run with:

```bash
docker-compose up -d
```

## Processing Logs in Real-Time

To process Traefik logs in real-time as they're being written:

```yaml
# config.yaml
input:
  file: "/logs/access.log"
output:
  directory: "/output"
```

## Processing Multiple Log Files

To process all log files in a directory:

```yaml
# config.yaml
input:
  directory: "/logs"
  pattern: "*.log"
output:
  directory: "/output"
```

## Reading from Stdin (Piping)

To read logs from stdin (useful for piping from other commands):

```yaml
# config.yaml
input:
  stdin: true
output:
  directory: "/output"
```

You can then pipe Traefik logs to the application:

```bash
cat /path/to/traefik.log | docker run -i --rm \
  -v /path/to/output:/output \
  -v /path/to/config.yaml:/app/config.yaml \
  ghcr.io/hhftechnology/traefik-log-processor:latest
```

## Custom Log Rotation

For custom log rotation settings:

```yaml
# config.yaml
rotation:
  max_size: 50 # 50 MB per file
  max_age: 14 # 14 days retention
  max_backups: 10 # Keep 10 backups
  compress: true # Compress old logs
```

## Integration with Traefik

### Docker Compose Integration

```yaml
version: "3"

services:
  traefik:
    image: traefik:v3.3.4
    command:
      - "--api.insecure=true"
      - "--providers.docker=true"
      - "--accesslog=true"
      - "--accesslog.filepath=/logs/access.log"
      - "--accesslog.format=json"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - traefik-logs:/logs
    ports:
      - "80:80"
      - "8080:8080"

  log-splitter:
    image: ghcr.io/hhftechnology/traefik-log-processor:latest
    volumes:
      - traefik-logs:/logs:ro
      - ./split-logs:/output

volumes:
  traefik-logs:
```

### Kubernetes Integration

When using Traefik in Kubernetes, you can deploy the log splitter as a sidecar container:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: traefik
spec:
  selector:
    matchLabels:
      app: traefik
  template:
    metadata:
      labels:
        app: traefik
    spec:
      containers:
        - name: traefik
          image: traefik:v3.3.4
          args:
            - "--api.insecure=true"
            - "--providers.kubernetescrd=true"
            - "--accesslog=true"
            - "--accesslog.filepath=/logs/access.log"
            - "--accesslog.format=json"
          volumeMounts:
            - name: logs
              mountPath: /logs

        - name: log-splitter
          image: ghcr.io/hhftechnology/traefik-log-processor:latest
          volumeMounts:
            - name: logs
              mountPath: /logs
              readOnly: true
            - name: split-logs
              mountPath: /output

      volumes:
        - name: logs
          emptyDir: {}
        - name: split-logs
          persistentVolumeClaim:
            claimName: traefik-split-logs
```

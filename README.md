
# Traefik Log Processor

This Docker Compose setup uses the `hhftechnology/traefik-log-processor` image to process Traefik logs, splitting them into separate folders based on the "ServiceName" field and implementing log rotation and retention.

## Introduction

The `hhftechnology/traefik-log-processor` image contains a Python script that reads Traefik's JSON-formatted logs from standard input, parses each log entry, and writes it to a file in a directory structure based on the "ServiceName" and the date. It also performs periodic cleanup to remove old log files based on the retention policy.

## Prerequisites

- Docker
- Docker Compose

## Usage

To use this log processor with your existing Traefik setup:

1. **Configure Traefik to write logs in JSON format to a file.** For example, in your Traefik configuration:

   ```yaml
   [log]
     filePath = "/logs/traefik.log"
     format = "json"
   ```

   And in your Traefik Docker Compose service:

   ```yaml
   services:
     traefik:
       image: traefik:v3.3.4
       volumes:
         - traefik_logs:/logs
       # ... other configurations ...
   ```

2. **Create a Docker Compose file for the log processor:**

   ```yaml
   services:
     log_processor:
       image: hhftechnology/traefik-log-processor
       volumes:
         - traefik_logs:/input_logs
         - processed_logs:/logs
       command: tail -F /input_logs/traefik.log | python /app/process_logs.py
   volumes:
     traefik_logs:
       external: true
     processed_logs:
   ```

   This assumes that `traefik_logs` is the volume where Traefik writes its log file.

3. **Start the log processor:**

   Run the following command in the directory containing your `docker-compose.yml` file:

   ```bash
   docker compose up -d
   ```

The processed logs will be written to the `processed_logs` volume, organized by service name and date, e.g., `/logs/<service_name>/<YYYY-MM-DD>.log`.

## Customization

You can customize the log processing behavior using environment variables:

- `LOG_DIR`: Directory where processed logs are written (default: `/logs`)
- `RETENTION_DAYS`: Number of days to retain logs (default: 30)
- `CLEANUP_INTERVAL_HOURS`: Interval in hours between cleanup operations (default: 1)

For example, to change the retention period to 7 days, update your `docker-compose.yml`:

```yaml
services:
  log_processor:
    image: hhftechnology/traefik-log-processor
    environment:
      - RETENTION_DAYS=7
    volumes:
      - traefik_logs:/input_logs
      - processed_logs:/logs
    command: tail -F /input_logs/traefik.log | python /app/process_logs.py
```

## Accessing Processed Logs

To access the processed logs from the host, you can mount the `processed_logs` volume to a host directory. For example:

```yaml
volumes:
  processed_logs:
    driver: local
    driver_opts:
      type: none
      o: bind
      device: /host/path/to/processed/logs
```

Then, the logs will be available at `/host/path/to/processed/logs/<service_name>/<YYYY-MM-DD>.log`.

## License

This project is licensed under the MIT License.

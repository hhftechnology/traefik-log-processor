import json
import os
import sys
import datetime
import time
import glob

# Configuration from environment variables
LOG_DIR = os.environ.get('LOG_DIR', '/logs')
RETENTION_DAYS = int(os.environ.get('RETENTION_DAYS', '30'))
CLEANUP_INTERVAL_HOURS = int(os.environ.get('CLEANUP_INTERVAL_HOURS', '1'))

# Initialize last cleanup time
last_cleanup_time = time.time()

def perform_cleanup():
    now = datetime.datetime.now(datetime.timezone.utc)
    cutoff_date = now - datetime.timedelta(days=RETENTION_DAYS)
    for service_folder in glob.glob(os.path.join(LOG_DIR, '*')):
        if os.path.isdir(service_folder):
            for log_file in glob.glob(os.path.join(service_folder, '*.log')):
                file_name = os.path.basename(log_file)
                try:
                    file_date_str = file_name.split('.')[0]
                    file_date = datetime.datetime.strptime(file_date_str, '%Y-%m-%d').replace(tzinfo=datetime.timezone.utc)
                    if file_date < cutoff_date:
                        os.remove(log_file)
                        print(f"Deleted old log file: {log_file}", file=sys.stderr)
                except Exception as e:
                    print(f"Error processing file {log_file}: {e}", file=sys.stderr)

for line in sys.stdin:
    try:
        log_entry = json.loads(line)
        service_name = log_entry.get('ServiceName', 'unknown')
        start_utc = log_entry.get('StartUTC', None)
        if start_utc:
            log_date = datetime.datetime.fromisoformat(start_utc.replace('Z', '+00:00')).date()
        else:
            log_date = datetime.date.today()
        folder_path = os.path.join(LOG_DIR, service_name)
        os.makedirs(folder_path, exist_ok=True)
        file_name = f"{log_date.isoformat()}.log"
        file_path = os.path.join(folder_path, file_name)
        with open(file_path, 'a') as f:
            f.write(line)
    except json.JSONDecodeError:
        print(f"Invalid JSON: {line}", file=sys.stderr)
    except Exception as e:
        print(f"Error processing log entry: {e}", file=sys.stderr)
    
    # Check if it's time for cleanup
    current_time = time.time()
    if current_time > last_cleanup_time + CLEANUP_INTERVAL_HOURS * 3600:
        print("Performing cleanup...", file=sys.stderr)
        perform_cleanup()
        last_cleanup_time = current_time

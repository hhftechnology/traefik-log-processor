FROM python:3.9-slim
WORKDIR /app
COPY process_logs.py .
CMD ["python", "process_logs.py"]

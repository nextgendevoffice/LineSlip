FROM python:3.11

RUN apt-get update && \
    apt-get install -y libzbar0

COPY requirements.txt /
RUN pip install --no-cache-dir -r /requirements.txt

COPY . /app
WORKDIR /app

CMD ["gunicorn", "main:app", "--bind", "0.0.0.0:8000", "--workers", "3"]

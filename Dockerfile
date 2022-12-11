FROM python:3-slim

# Magic python/pip environment variables
ENV PYTHONUNBUFFERED=1 \
  PYTHONDONTWRITEBYTECODE=1 \
  PYTHONHASHSEED=random \
  PIP_NO_CACHE_DIR=off \
  PIP_DISABLE_PIP_VERSION_CHECK=on

RUN pip install poetry

WORKDIR /etc/adventofcode

COPY pyproject.toml poetry.lock ./
RUN apt-get update && apt-get install -y build-essential \
    && poetry install \
    && apt-get remove -y --purge build-essential \
    && apt-get autoremove -y \
    && rm -rf /var/lib/apt/lists/*
COPY . .

CMD ["poetry", "run", "python", "-m", "adventofcode"]

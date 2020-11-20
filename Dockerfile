FROM python:3-slim

# Magic python/pip environment variables
ENV PYTHONUNBUFFERED=1 \
  PYTHONDONTWRITEBYTECODE=1 \
  PYTHONHASHSEED=random \
  PIP_NO_CACHE_DIR=off \
  PIP_DISABLE_PIP_VERSION_CHECK=on

RUN pip install poetry

WORKDIR /etc/adventofcode

COPY pyproject.toml poetry.lock .
RUN poetry install
COPY . .

CMD ["python", "-m", "adventofcode"]

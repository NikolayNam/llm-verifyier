FROM python:3.14.3-slim-trixie

WORKDIR /app

COPY --from=ghcr.io/astral-sh/uv:0.11.2 /uv /uvx /bin/

WORKDIR /app
COPY requirements.txt .

ENV UV_SYSTEM_PYTHON=1
ENV UV_LINK_MODE=copy

RUN --mount=type=cache,target=/root/.cache/uv \
    uv pip install --system -r requirements.txt

COPY . .

CMD ["python", "benchmark_runner.py"]
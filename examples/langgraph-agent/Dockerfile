FROM python:3.12-slim
WORKDIR /app

RUN pip install --no-cache-dir uv

COPY pyproject.toml uv.lock ./
RUN uv sync --locked

COPY main.py ./

EXPOSE 9002
CMD ["uv", "run", "main.py"]

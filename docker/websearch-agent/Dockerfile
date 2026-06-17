FROM python:3.11-slim

WORKDIR /app

RUN pip install httpx pydantic pydantic-ai

COPY agent.py /app/
COPY pyproject.toml /app/

RUN pip install -e .

EXPOSE 8082

CMD ["python", "agent.py"]
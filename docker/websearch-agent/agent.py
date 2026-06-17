"""Web search agent for research tasks.

Uses Qwen3.6 via HTTP to perform web searches and summarize results.
"""

import os
from dataclasses import dataclass

import httpx


LLM_ENDPOINT = os.environ.get("LLM_ENDPOINT", "http://localhost:8080/v1")
LLM_API_KEY = os.environ.get("LLM_API_KEY", "test-key")
USER_ID = os.environ.get("USER_ID", "soypete")


@dataclass
class SearchResult:
    title: str
    url: str
    snippet: str


class WebSearchAgent:
    """Agent that uses LLM to search the web for information."""

    def __init__(self, endpoint: str = LLM_ENDPOINT, api_key: str = LLM_API_KEY):
        self.endpoint = endpoint.rstrip("/")
        self.api_key = api_key
        self.client = httpx.Client(timeout=60.0)

    def search(self, query: str) -> list[SearchResult]:
        """Search for information using the LLM as a search proxy."""
        prompt = f"""Given the user query: "{query}"

Search the web for relevant information. Return a list of results with:
- title
- url (if found)
- brief snippet

Format as JSON array:
[{{"title": "...", "url": "...", "snippet": "..."}}]"""

        try:
            response = self.client.post(
                f"{self.endpoint}/chat/completions",
                headers={"Authorization": f"Bearer {self.api_key}"},
                json={
                    "model": "qwen3.6",
                    "messages": [{"role": "user", "content": prompt}],
                    "temperature": 0.3,
                },
            )
            response.raise_for_status()
            content = response.json()["choices"][0]["message"]["content"]

            import json

            results = json.loads(content)
            return [SearchResult(**r) for r in results]
        except Exception as e:
            return [
                SearchResult(
                    title=f"Error: {str(e)}",
                    url="",
                    snippet="Failed to search",
                )
            ]

    def research(self, topic: str, depth: int = 3) -> str:
        """Perform in-depth research on a topic."""
        results = self.search(topic)
        summary_prompt = f"""Research topic: {topic}

Results found:
{chr(10).join([f"- {r.title}: {r.snippet}" for r in results])}

Provide a comprehensive summary with citations."""

        try:
            response = self.client.post(
                f"{self.endpoint}/chat/completions",
                headers={"Authorization": f"Bearer {self.api_key}"},
                json={
                    "model": "qwen3.6",
                    "messages": [{"role": "user", "content": summary_prompt}],
                    "temperature": 0.5,
                },
            )
            response.raise_for_status()
            return response.json()["choices"][0]["message"]["content"]
        except Exception as e:
            return f"Research failed: {e}"

    def close(self) -> None:
        self.client.close()


if __name__ == "__main__":
    agent = WebSearchAgent()

    query = "soypete tech github"
    print(f"Searching for: {query}")
    results = agent.search(query)
    for r in results:
        print(f"  - {r.title}: {r.snippet}")

    print("\n--- Research ---")
    summary = agent.research("who is soypete tech")
    print(summary)

    agent.close()
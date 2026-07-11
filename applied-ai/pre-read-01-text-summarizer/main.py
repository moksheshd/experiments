"""Summarize a long text file using Gemini on Google Vertex AI.

Usage:
    python main.py [path/to/file.txt]

Reads the given text file (defaults to input.txt), sends it to Gemini via
Vertex AI, and prints the summary to stdout. By default it also renders the
request (system prompt + input) and response (token usage + summary) as rich
panels on stderr — pass --quiet to hide the panels.

Authentication uses Google Application Default Credentials (ADC) — run
`gcloud auth application-default login` once. No API key is needed.

Configuration is read from environment variables, each with a fallback so a
normal run needs nothing exported:
    GOOGLE_CLOUD_PROJECT    GCP project ID (falls back to the active gcloud project)
    GOOGLE_CLOUD_LOCATION   Vertex location (falls back to "global")
"""

import argparse
import os
import subprocess
import sys

from google import genai
from google.genai import types
from rich.console import Console
from rich.panel import Panel
from rich.text import Text

MODEL = "gemini-2.5-flash-lite"
MAX_TOKENS = 1024

SYSTEM_PROMPT = (
    "You are a precise summarizer. Produce a clear, faithful summary "
    "of the user's text. Lead with a one-sentence overview, then give "
    "the key points as concise bullets. Do not add information that is "
    "not in the source."
)


def gcloud_default_project() -> str | None:
    """Return the active gcloud project, so GOOGLE_CLOUD_PROJECT need not be set.

    Falls back to `gcloud config get-value project`; returns None if gcloud is
    missing or no project is configured. Resolved per-process — nothing exported.
    """
    try:
        result = subprocess.run(
            ["gcloud", "config", "get-value", "project"],
            capture_output=True,
            text=True,
            check=False,
        )
    except FileNotFoundError:
        return None
    project = result.stdout.strip()
    return project or None


def render_request(console: Console, system_prompt: str, text: str) -> None:
    """Pretty-print the outgoing request: system prompt + the input text."""
    preview = text if len(text) <= 800 else text[:800] + f"\n… [+{len(text) - 800:,} more chars]"
    body = Text.assemble(
        ("model:  ", "bold"), f"{MODEL}\n",
        ("tokens: ", "bold"), f"max_output_tokens={MAX_TOKENS}\n\n",
        ("system_instruction:\n", "bold"), (system_prompt, "italic"),
        ("\n\ninput:\n", "bold"), preview,
    )
    console.print(Panel(body, title="[bold cyan]→ Request[/]", border_style="cyan"))


def render_response(console: Console, summary: str, usage) -> None:
    """Pretty-print the incoming response: token usage + the summary."""
    body = Text.assemble(
        ("tokens in/out: ", "bold"),
        f"{usage.prompt_token_count}/{usage.candidates_token_count}\n\n",
        ("summary:\n", "bold"), summary,
    )
    console.print(Panel(body, title="[bold green]← Response[/]", border_style="green"))


def read_text(path: str) -> str:
    try:
        with open(path, "r", encoding="utf-8") as f:
            text = f.read().strip()
    except FileNotFoundError:
        sys.exit(f"Error: file not found: {path}")

    if not text:
        sys.exit(f"Error: file is empty: {path}")
    return text


def summarize(client: genai.Client, console: Console | None, text: str) -> str:
    if console is not None:
        render_request(console, SYSTEM_PROMPT, text)
    response = client.models.generate_content(
        model=MODEL,
        contents=f"Summarize the following text:\n\n{text}",
        config=types.GenerateContentConfig(
            system_instruction=SYSTEM_PROMPT,
            max_output_tokens=MAX_TOKENS,
        ),
    )
    summary = response.text
    if console is not None:
        render_response(console, summary, response.usage_metadata)
    return summary


def main() -> None:
    parser = argparse.ArgumentParser(description="Summarize a text file with Gemini on Vertex AI.")
    parser.add_argument(
        "path",
        nargs="?",
        default="input.txt",
        help="Path to the text file to summarize (default: input.txt)",
    )
    parser.add_argument(
        "--quiet",
        action="store_true",
        help="Hide the request/response panels; print only the summary.",
    )
    args = parser.parse_args()

    # Rich panels go to stderr so stdout stays just the summary; --quiet turns them off.
    console = None if args.quiet else Console(stderr=True)

    # Env vars win; otherwise fall back to the active gcloud project and the
    # "global" location, so nothing has to be exported for a normal run.
    project_id = os.environ.get("GOOGLE_CLOUD_PROJECT") or gcloud_default_project()
    location = os.environ.get("GOOGLE_CLOUD_LOCATION") or "global"
    if not project_id:
        sys.exit(
            "Error: no GCP project found. Set GOOGLE_CLOUD_PROJECT or run "
            "`gcloud config set project <project-id>`."
        )

    text = read_text(args.path)
    # Point the google-genai client at the Vertex AI backend. It reads project
    # and location from the env vars above; we pass them explicitly so a
    # misconfiguration fails fast with a clear message.
    client = genai.Client(vertexai=True, project=project_id, location=location)

    print(
        f"Summarizing {args.path} ({len(text):,} chars) with {MODEL} "
        f"on Vertex AI [{project_id} / {location}]...\n",
        file=sys.stderr,
    )
    summary = summarize(client, console, text)
    print(summary)


if __name__ == "__main__":
    main()

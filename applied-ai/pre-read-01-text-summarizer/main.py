"""Summarize a long text file using Gemini on Google Vertex AI.

Usage:
    python main.py [path/to/file.txt]

Reads the given text file (defaults to input.txt), sends it to Gemini via
Vertex AI, and prints the summary to stdout.

Authentication uses Google Application Default Credentials (ADC) — run
`gcloud auth application-default login` once. No API key is needed.

Configuration is read from environment variables:
    GOOGLE_CLOUD_PROJECT    GCP project ID hosting the Gemini models
    GOOGLE_CLOUD_LOCATION   Vertex location (e.g. "global", "us-central1")
"""

import argparse
import os
import subprocess
import sys

from google import genai
from google.genai import types

MODEL = "gemini-2.5-flash-lite"
MAX_TOKENS = 1024

SYSTEM_PROMPT = (
    "You are a precise summarizer. Produce a clear, faithful summary "
    "of the user's text. Lead with a one-sentence overview, then give "
    "the key points as concise bullets. Do not add information that is "
    "not in the source."
)


def read_text(path: str) -> str:
    try:
        with open(path, "r", encoding="utf-8") as f:
            text = f.read().strip()
    except FileNotFoundError:
        sys.exit(f"Error: file not found: {path}")

    if not text:
        sys.exit(f"Error: file is empty: {path}")
    return text


def summarize(client: genai.Client, text: str) -> str:
    response = client.models.generate_content(
        model=MODEL,
        contents=f"Summarize the following text:\n\n{text}",
        config=types.GenerateContentConfig(
            system_instruction=SYSTEM_PROMPT,
            max_output_tokens=MAX_TOKENS,
        ),
    )
    return response.text


def main() -> None:
    parser = argparse.ArgumentParser(description="Summarize a text file with Gemini on Vertex AI.")
    parser.add_argument(
        "path",
        nargs="?",
        default="input.txt",
        help="Path to the text file to summarize (default: input.txt)",
    )
    args = parser.parse_args()

    project_id = os.environ.get("GOOGLE_CLOUD_PROJECT")
    location = os.environ.get("GOOGLE_CLOUD_LOCATION")
    if not project_id:
        sys.exit("Error: GOOGLE_CLOUD_PROJECT environment variable is not set.")
    if not location:
        sys.exit("Error: GOOGLE_CLOUD_LOCATION environment variable is not set.")

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
    summary = summarize(client, text)
    print(summary)


if __name__ == "__main__":
    main()

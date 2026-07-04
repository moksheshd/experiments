"""Summarize a long text file using Claude on Google Vertex AI.

Usage:
    python summarize.py [path/to/file.txt]

Reads the given text file (defaults to input.txt), sends it to Claude via
Vertex AI, and prints the summary to stdout.

Authentication uses Google Application Default Credentials (ADC) — run
`gcloud auth application-default login` once. No Anthropic API key is needed.

Configuration is read from environment variables:
    ANTHROPIC_VERTEX_PROJECT_ID   GCP project ID hosting the Claude models
    CLOUD_ML_REGION               Vertex region (e.g. "global", "us-east5")
"""

import argparse
import os
import sys

from anthropic import AnthropicVertex

MODEL = "claude-opus-4-8"
MAX_TOKENS = 1024


def read_text(path: str) -> str:
    try:
        with open(path, "r", encoding="utf-8") as f:
            text = f.read().strip()
    except FileNotFoundError:
        sys.exit(f"Error: file not found: {path}")

    if not text:
        sys.exit(f"Error: file is empty: {path}")
    return text


def summarize(client: AnthropicVertex, text: str) -> str:
    message = client.messages.create(
        model=MODEL,
        max_tokens=MAX_TOKENS,
        system=(
            "You are a precise summarizer. Produce a clear, faithful summary "
            "of the user's text. Lead with a one-sentence overview, then give "
            "the key points as concise bullets. Do not add information that is "
            "not in the source."
        ),
        messages=[
            {
                "role": "user",
                "content": f"Summarize the following text:\n\n{text}",
            }
        ],
    )
    return "".join(block.text for block in message.content if block.type == "text")


def main() -> None:
    parser = argparse.ArgumentParser(description="Summarize a text file with Claude on Vertex AI.")
    parser.add_argument(
        "path",
        nargs="?",
        default="input.txt",
        help="Path to the text file to summarize (default: input.txt)",
    )
    args = parser.parse_args()

    project_id = os.environ.get("ANTHROPIC_VERTEX_PROJECT_ID")
    region = os.environ.get("CLOUD_ML_REGION")
    if not project_id:
        sys.exit("Error: ANTHROPIC_VERTEX_PROJECT_ID environment variable is not set.")
    if not region:
        sys.exit("Error: CLOUD_ML_REGION environment variable is not set.")

    text = read_text(args.path)
    # AnthropicVertex reads project_id/region from the env vars above; we pass
    # them explicitly so a misconfiguration fails fast with a clear message.
    client = AnthropicVertex(project_id=project_id, region=region)

    print(
        f"Summarizing {args.path} ({len(text):,} chars) with {MODEL} "
        f"on Vertex AI [{project_id} / {region}]...\n",
        file=sys.stderr,
    )
    summary = summarize(client, text)
    print(summary)


if __name__ == "__main__":
    main()

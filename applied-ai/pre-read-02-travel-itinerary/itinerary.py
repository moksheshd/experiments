"""Generate a structured travel itinerary using Claude on Google Vertex AI.

Usage:
    python itinerary.py [--seed N] [--quiet]

Picks a random source and destination city, asks Claude to plan the trip from
its own knowledge, and prints a JSON itinerary that conforms to a fixed Pydantic
schema.

Structured output is enforced with tool use: the Pydantic schema is generated
with `Itinerary.model_json_schema()` and handed to Claude as a tool, `tool_choice`
forces Claude to reply with arguments matching that schema, and the tool input is
validated back into a Pydantic model with `Itinerary.model_validate(...)`.

Progress is logged to stderr so you can follow each step; the final itinerary is
the only thing written to stdout. By default (learning mode) it logs the full
detail — the generated JSON schema, the request and response payloads, and the
raw tool input; pass --quiet to show only the INFO-level progress summary.

Authentication uses Google Application Default Credentials (ADC) — run
`gcloud auth application-default login` once. No Anthropic API key is needed.

Configuration is read from environment variables:
    ANTHROPIC_VERTEX_PROJECT_ID   GCP project ID hosting the Claude models
    CLOUD_ML_REGION               Vertex region (e.g. "global", "us-east5")
"""

import argparse
import json
import logging
import os
import random
import sys

from anthropic import AnthropicVertex
from pydantic import BaseModel

MODEL = "claude-opus-4-8"
MAX_TOKENS = 2048
TOOL_NAME = "save_itinerary"

# Logs go to stderr; stdout is reserved for the final JSON itinerary.
log = logging.getLogger("itinerary")

CITIES = [
    "London",
    "Tokyo",
    "New York",
    "Paris",
    "Sydney",
    "Cape Town",
    "Reykjavik",
    "Buenos Aires",
    "Bangkok",
    "Vancouver",
    "Lisbon",
    "Marrakech",
    "Kyoto",
    "Istanbul",
    "Queenstown",
]


class DayPlan(BaseModel):
    day: int
    activities: list[str]


class Itinerary(BaseModel):
    destination: str
    trip_duration_days: int
    budget_category: str
    top_attractions: list[str]
    daily_plan: list[DayPlan]


def pick_cities() -> tuple[str, str]:
    source, destination = random.sample(CITIES, 2)
    log.info("Picked cities from a pool of %d: %s -> %s", len(CITIES), source, destination)
    return source, destination


def generate_itinerary(client: AnthropicVertex, source: str, destination: str) -> Itinerary:
    # Step 1: turn the Pydantic model into a JSON Schema and wrap it as a tool.
    # The tool's input_schema IS the Pydantic schema.
    schema = Itinerary.model_json_schema()
    log.info("Built JSON schema from the Itinerary model (fields: %s)", ", ".join(schema["required"]))
    log.debug("Full tool input_schema:\n%s", json.dumps(schema, indent=2))

    tool = {
        "name": TOOL_NAME,
        "description": "Record a complete travel itinerary for the trip.",
        "input_schema": schema,
    }

    # Step 2: build the request. Forcing tool_choice makes Claude answer by
    # "calling" the tool, so its reply must match the schema instead of free text.
    # We assemble the payload as one dict so we can both log it and send it.
    request = {
        "model": MODEL,
        "max_tokens": MAX_TOKENS,
        "tools": [tool],
        "tool_choice": {"type": "tool", "name": TOOL_NAME},
        "messages": [
            {
                "role": "user",
                "content": (
                    f"Plan a trip from {source} to {destination}. Choose a sensible "
                    "trip length and budget category, and use your own knowledge of the "
                    f"destination. Call the {TOOL_NAME} tool with the complete itinerary."
                ),
            }
        ],
    }
    log.info("Calling %s with forced tool_choice=%s ...", MODEL, TOOL_NAME)
    log.debug("Request payload sent to Claude:\n%s", json.dumps(request, indent=2))

    message = client.messages.create(**request)

    log.debug("Response payload from Claude:\n%s", message.model_dump_json(indent=2))
    log.info(
        "Response received: stop_reason=%s, blocks=%s, tokens in/out=%d/%d",
        message.stop_reason,
        [block.type for block in message.content],
        message.usage.input_tokens,
        message.usage.output_tokens,
    )

    # Step 3: pull the tool_use block out of the response; its .input is a dict.
    tool_use = next((block for block in message.content if block.type == "tool_use"), None)
    if tool_use is None:
        sys.exit("Error: model did not return a tool_use block.")
    log.info("Found tool_use block for '%s'", tool_use.name)
    log.debug("Raw tool input (unvalidated dict):\n%s", json.dumps(tool_use.input, indent=2))

    # Step 4: validate + coerce the raw tool input into a typed Pydantic model.
    # This is where a missing field or wrong type would raise.
    itinerary = Itinerary.model_validate(tool_use.input)
    log.info(
        "Validated into Itinerary: %d-day trip to %s (%s), %d attractions, %d daily entries",
        itinerary.trip_duration_days,
        itinerary.destination,
        itinerary.budget_category,
        len(itinerary.top_attractions),
        len(itinerary.daily_plan),
    )
    return itinerary


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Generate a structured travel itinerary with Claude on Vertex AI."
    )
    parser.add_argument(
        "--seed",
        type=int,
        default=None,
        help="Optional random seed to make the city selection reproducible.",
    )
    parser.add_argument(
        "--quiet",
        action="store_true",
        help="Show only INFO-level progress (hide schema, request/response, raw input).",
    )
    args = parser.parse_args()

    logging.basicConfig(
        level=logging.INFO if args.quiet else logging.DEBUG,
        format="%(asctime)s %(levelname)-5s %(message)s",
        datefmt="%H:%M:%S",
        stream=sys.stderr,
    )

    if args.seed is not None:
        random.seed(args.seed)
        log.info("Random seed set to %d (reproducible city selection)", args.seed)

    project_id = os.environ.get("ANTHROPIC_VERTEX_PROJECT_ID")
    region = os.environ.get("CLOUD_ML_REGION")
    if not project_id:
        sys.exit("Error: ANTHROPIC_VERTEX_PROJECT_ID environment variable is not set.")
    if not region:
        sys.exit("Error: CLOUD_ML_REGION environment variable is not set.")

    source, destination = pick_cities()
    # AnthropicVertex reads project_id/region from the env vars above; we pass
    # them explicitly so a misconfiguration fails fast with a clear message.
    log.info("Creating AnthropicVertex client [project=%s, region=%s]", project_id, region)
    client = AnthropicVertex(project_id=project_id, region=region)

    itinerary = generate_itinerary(client, source, destination)

    log.info("Writing itinerary JSON to stdout")
    print(itinerary.model_dump_json(indent=2))


if __name__ == "__main__":
    main()

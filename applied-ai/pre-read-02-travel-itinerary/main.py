"""Generate a structured travel itinerary using Gemini on Google Vertex AI.

Usage:
    python main.py [--seed N] [--quiet]

Picks a random source and destination city, asks Gemini to plan the trip from
its own knowledge, and prints a JSON itinerary that conforms to a fixed Pydantic
schema.

Structured output is enforced with Gemini's native JSON mode: the request sets
`response_mime_type="application/json"` and hands the Pydantic model directly as
`response_schema`, so the model must reply with JSON matching that schema. The
SDK then parses and validates the reply back into a typed model, exposed as
`response.parsed`.

Progress is logged to stderr so you can follow each step; the final itinerary is
the only thing written to stdout. By default (learning mode) it logs the full
detail — the generated JSON schema, the request and response payloads, and the
raw JSON text — pass --quiet to show only the INFO-level progress summary.

Authentication uses Google Application Default Credentials (ADC) — run
`gcloud auth application-default login` once. No API key is needed.

Configuration is read from environment variables:
    GOOGLE_CLOUD_PROJECT    GCP project ID hosting the Gemini models
    GOOGLE_CLOUD_LOCATION   Vertex location (e.g. "global", "us-central1")
"""

import argparse
import json
import logging
import os
import random
import sys

from google import genai
from google.genai import types
from pydantic import BaseModel

MODEL = "gemini-2.5-flash-lite"
MAX_TOKENS = 2048

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


def generate_itinerary(client: genai.Client, source: str, destination: str) -> Itinerary:
    # Step 1: the Pydantic model IS the response schema. Log the JSON Schema it
    # produces so you can see the shape Gemini is being constrained to.
    schema = Itinerary.model_json_schema()
    log.info("Built JSON schema from the Itinerary model (fields: %s)", ", ".join(schema["required"]))
    log.debug("Full response schema:\n%s", json.dumps(schema, indent=2))

    # Step 2: build the request. Setting response_mime_type + response_schema puts
    # Gemini in structured-output mode, so its reply must be JSON matching the
    # schema instead of free text.
    prompt = (
        f"Plan a trip from {source} to {destination}. Choose a sensible "
        "trip length and budget category, and use your own knowledge of the "
        "destination. Return the complete itinerary as JSON."
    )
    config = types.GenerateContentConfig(
        response_mime_type="application/json",
        response_schema=Itinerary,
        max_output_tokens=MAX_TOKENS,
    )
    log.info("Calling %s with response_schema=Itinerary ...", MODEL)
    log.debug("Prompt sent to Gemini:\n%s", prompt)

    response = client.models.generate_content(model=MODEL, contents=prompt, config=config)

    log.debug("Response payload from Gemini:\n%s", response.model_dump_json(indent=2))
    usage = response.usage_metadata
    log.info(
        "Response received: finish_reason=%s, tokens in/out=%d/%d",
        response.candidates[0].finish_reason,
        usage.prompt_token_count,
        usage.candidates_token_count,
    )

    # Step 3: the raw JSON text is what the model actually emitted; log it before
    # the SDK validates it.
    log.debug("Raw JSON text (unvalidated):\n%s", response.text)

    # Step 4: response.parsed is the JSON already validated + coerced into the
    # typed Pydantic model. This is where a missing field or wrong type would raise.
    itinerary = response.parsed
    if not isinstance(itinerary, Itinerary):
        sys.exit("Error: model did not return a valid Itinerary.")
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
        description="Generate a structured travel itinerary with Gemini on Vertex AI."
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
        help="Show only INFO-level progress (hide schema, request/response, raw JSON).",
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

    project_id = os.environ.get("GOOGLE_CLOUD_PROJECT")
    location = os.environ.get("GOOGLE_CLOUD_LOCATION")
    if not project_id:
        sys.exit("Error: GOOGLE_CLOUD_PROJECT environment variable is not set.")
    if not location:
        sys.exit("Error: GOOGLE_CLOUD_LOCATION environment variable is not set.")

    source, destination = pick_cities()
    # Point the google-genai client at the Vertex AI backend. It reads project
    # and location from the env vars above; we pass them explicitly so a
    # misconfiguration fails fast with a clear message.
    log.info("Creating google-genai Vertex client [project=%s, location=%s]", project_id, location)
    client = genai.Client(vertexai=True, project=project_id, location=location)

    itinerary = generate_itinerary(client, source, destination)

    log.info("Writing itinerary JSON to stdout")
    print(itinerary.model_dump_json(indent=2))


if __name__ == "__main__":
    main()

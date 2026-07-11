# pre-read-02: Travel Itinerary Generator

Generate a **structured** travel itinerary as JSON using Claude on **Google Vertex AI**,
via the official [Anthropic Python SDK](https://github.com/anthropics/anthropic-sdk-python)
and [Pydantic](https://docs.pydantic.dev/).

## What it does

Picks a random source and destination city, asks Claude to plan the trip from its own
knowledge, and prints a JSON itinerary that conforms to a fixed [Pydantic](https://docs.pydantic.dev/)
schema.

The schema is enforced with **tool use**: the Pydantic model is turned into a JSON Schema
(`Itinerary.model_json_schema()`) and handed to Claude as a tool, `tool_choice` forces
Claude to reply with arguments matching that schema, and the tool input is validated back
into a typed model with `Itinerary.model_validate(...)`.

The output always has these fields:

| Field                | Type                                        |
| -------------------- | ------------------------------------------- |
| `destination`        | `str`                                       |
| `trip_duration_days` | `int`                                       |
| `budget_category`    | `str`                                       |
| `top_attractions`    | `list[str]`                                 |
| `daily_plan`         | `list[{day: int, activities: list[str]}]`   |

## Setup

```bash
# From this directory
python -m venv .venv                # create the virtual environment
source .venv/bin/activate           # activate it
pip install -r requirements.txt     # install deps
```

Authenticate with Google Application Default Credentials (once):

```bash
gcloud auth application-default login
```

Set the Vertex configuration (these are read by the SDK's `AnthropicVertex` client):

```bash
export ANTHROPIC_VERTEX_PROJECT_ID="your-gcp-project-id"
export CLOUD_ML_REGION="global"        # or a specific region, e.g. us-east5
```

No Anthropic API key is required — auth flows through GCP ADC.

## Usage

```bash
# Random source and destination each run.
# Full detail is logged by default (learning mode): the generated JSON schema,
# the request and response payloads, and the raw tool input before validation.
python itinerary.py

# Reproducible city selection
python itinerary.py --seed 42

# Show only the INFO-level progress summary (hide the detailed dumps)
python itinerary.py --quiet
```

Progress is logged to stderr (so `stdout` stays pure JSON). To capture just the
itinerary, redirect stdout: `python itinerary.py > trip.json`.

## Files

| File               | Purpose                                             |
| ------------------ | --------------------------------------------------- |
| `itinerary.py`     | Main script: pick cities → call Claude → validate → print |
| `requirements.txt` | Pinned dependencies (`anthropic[vertex]`, `pydantic`) |
| `.venv/`           | Local virtual environment (gitignored)              |

## Notes

- Model is set via the `MODEL` constant in `itinerary.py` (default: `claude-opus-4-8`).
  The model must be enabled in your Vertex project/region.
- On Vertex AI, model IDs use the bare first-party string (no `anthropic.` prefix).
- Project and region are read from `ANTHROPIC_VERTEX_PROJECT_ID` and
  `CLOUD_ML_REGION`; nothing is hardcoded or committed.
- The Pydantic schema is the single source of truth for the output shape — change the
  `Itinerary` / `DayPlan` models and the tool schema and validation follow automatically.
- The SDK also offers `client.messages.parse(output_format=Itinerary)` as a one-line
  shortcut that hides the schema-generation and validation steps used here.

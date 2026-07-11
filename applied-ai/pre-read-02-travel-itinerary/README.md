# pre-read-02: Travel Itinerary Generator

Generate a **structured** travel itinerary as JSON using Gemini on **Google Vertex AI**,
via the official [Google Gen AI SDK](https://github.com/googleapis/python-genai)
and [Pydantic](https://docs.pydantic.dev/).

## What it does

Picks a random source and destination city, asks Gemini to plan the trip from its own
knowledge, and prints a JSON itinerary that conforms to a fixed [Pydantic](https://docs.pydantic.dev/)
schema.

The schema is enforced with Gemini's **native structured output**: the request sets
`response_mime_type="application/json"` and passes the Pydantic model directly as
`response_schema`, so Gemini must reply with JSON matching that schema. The SDK then parses
and validates the reply back into a typed model, exposed as `response.parsed`.

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

Set the Vertex configuration (these point the `google-genai` client at the Vertex backend):

```bash
export GOOGLE_CLOUD_PROJECT="your-gcp-project-id"
export GOOGLE_CLOUD_LOCATION="global"        # or a specific location, e.g. us-central1
```

No API key is required — auth flows through GCP ADC.

## Usage

```bash
# Random source and destination each run.
# Full detail is logged by default (learning mode): the generated JSON schema,
# the request and response payloads, and the raw JSON before validation.
python main.py

# Reproducible city selection
python main.py --seed 42

# Show only the INFO-level progress summary (hide the detailed dumps)
python main.py --quiet
```

Progress is logged to stderr (so `stdout` stays pure JSON). To capture just the
itinerary, redirect stdout: `python main.py > trip.json`.

## Files

| File               | Purpose                                             |
| ------------------ | --------------------------------------------------- |
| `main.py`          | Main script: pick cities → call Gemini → validate → print |
| `requirements.txt` | Pinned dependencies (`google-genai`, `pydantic`)    |
| `.venv/`           | Local virtual environment (gitignored)              |

## Notes

- Model is set via the `MODEL` constant in `main.py` (default: `gemini-2.5-flash-lite`).
  The model must be enabled in your Vertex project/location.
- Project and location are read from `GOOGLE_CLOUD_PROJECT` and
  `GOOGLE_CLOUD_LOCATION`; nothing is hardcoded or committed.
- The Pydantic schema is the single source of truth for the output shape — change the
  `Itinerary` / `DayPlan` models and the response schema and validation follow automatically.
- `response.parsed` returns the JSON already validated into a typed `Itinerary`; the raw
  JSON string is also available as `response.text`.

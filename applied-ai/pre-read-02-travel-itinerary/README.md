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

By default the script uses your **active gcloud project** and the `global` location, so no
extra config is needed. To override for a single run (without exporting), prefix the command:

```bash
GOOGLE_CLOUD_PROJECT="your-gcp-project-id" GOOGLE_CLOUD_LOCATION="us-central1" python main.py
```

No API key is required — auth flows through GCP ADC.

## Usage

```bash
# Random source and destination each run.
# By default (learning mode) the request (prompt + response schema) and the
# response (token usage + JSON body) are rendered as rich panels on stderr.
python main.py

# Reproducible city selection
python main.py --seed 42

# Show only the INFO-level progress summary (hide the request/response panels)
python main.py --quiet
```

Progress is logged to stderr (so `stdout` stays pure JSON). To capture just the
itinerary, redirect stdout: `python main.py > trip.json`.

## Files

| File               | Purpose                                             |
| ------------------ | --------------------------------------------------- |
| `main.py`          | Main script: pick cities → call Gemini → validate → print |
| `requirements.txt` | Pinned dependencies (`google-genai`, `pydantic`, `rich`) |
| `.venv/`           | Local virtual environment (gitignored)              |

## Notes

- Model is set via the `MODEL` constant in `main.py` (default: `gemini-2.5-flash-lite`).
  The model must be enabled in your Vertex project/location.
- Project and location come from `GOOGLE_CLOUD_PROJECT` / `GOOGLE_CLOUD_LOCATION` when set,
  otherwise from the active gcloud project and `global`; nothing is hardcoded or committed.
- The Pydantic schema is the single source of truth for the output shape — change the
  `Itinerary` / `DayPlan` models and the response schema and validation follow automatically.
- `response.parsed` returns the JSON already validated into a typed `Itinerary`; the raw
  JSON string is also available as `response.text`.

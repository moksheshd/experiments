# pre-read-01: Text Summarizer

Summarize a long text file using Gemini on **Google Vertex AI**, via the official
[Google Gen AI SDK](https://github.com/googleapis/python-genai).

## What it does

Reads a text file, sends its contents to Gemini (through Vertex AI), and prints a
concise summary (one-sentence overview + key bullets) to stdout.

## Setup

```bash
# From this directory
source .venv/bin/activate          # activate the virtual environment
pip install -r requirements.txt    # install deps (already installed in .venv)
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
# Summarize the bundled sample (input.txt)
# The request (system prompt + input) and response (token usage + summary)
# are rendered as rich panels on stderr; the summary itself goes to stdout.
python main.py

# Summarize your own file
python main.py path/to/your/file.txt

# Hide the panels; print only the summary
python main.py --quiet
```

## Files

| File               | Purpose                                        |
| ------------------ | ---------------------------------------------- |
| `main.py`          | Main script: read file → call Gemini → print   |
| `input.txt`        | Sample long text (ISRO Chandrayaan-3 mission)  |
| `requirements.txt` | Pinned dependencies (`google-genai`, `rich`)   |
| `.venv/`           | Local virtual environment (gitignored)         |

## Notes

- Model is set via the `MODEL` constant in `main.py` (default: `gemini-2.5-flash-lite`).
  The model must be enabled in your Vertex project/location.
- Project and location come from `GOOGLE_CLOUD_PROJECT` / `GOOGLE_CLOUD_LOCATION` when set,
  otherwise from the active gcloud project and `global`; nothing is hardcoded or committed.

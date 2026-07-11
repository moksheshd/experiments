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

Set the Vertex configuration (these point the `google-genai` client at the Vertex backend):

```bash
export GOOGLE_CLOUD_PROJECT="your-gcp-project-id"
export GOOGLE_CLOUD_LOCATION="global"        # or a specific location, e.g. us-central1
```

No API key is required — auth flows through GCP ADC.

## Usage

```bash
# Summarize the bundled sample (input.txt)
python main.py

# Summarize your own file
python main.py path/to/your/file.txt
```

## Files

| File               | Purpose                                        |
| ------------------ | ---------------------------------------------- |
| `main.py`          | Main script: read file → call Gemini → print   |
| `input.txt`        | Sample long text (ISRO Chandrayaan-3 mission)  |
| `requirements.txt` | Pinned dependencies (`google-genai`)           |
| `.venv/`           | Local virtual environment (gitignored)         |

## Notes

- Model is set via the `MODEL` constant in `main.py` (default: `gemini-2.5-flash-lite`).
  The model must be enabled in your Vertex project/location.
- Project and location are read from `GOOGLE_CLOUD_PROJECT` and
  `GOOGLE_CLOUD_LOCATION`; nothing is hardcoded or committed.

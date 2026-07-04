# pre-read-01: Text Summarizer

Summarize a long text file using Claude on **Google Vertex AI**, via the official
[Anthropic Python SDK](https://github.com/anthropics/anthropic-sdk-python).

## What it does

Reads a text file, sends its contents to Claude (through Vertex AI), and prints a
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

Set the Vertex configuration (these are read by the SDK's `AnthropicVertex` client):

```bash
export ANTHROPIC_VERTEX_PROJECT_ID="your-gcp-project-id"
export CLOUD_ML_REGION="global"        # or a specific region, e.g. us-east5
```

No Anthropic API key is required — auth flows through GCP ADC.

## Usage

```bash
# Summarize the bundled sample (input.txt)
python summarize.py

# Summarize your own file
python summarize.py path/to/your/file.txt
```

## Files

| File               | Purpose                                        |
| ------------------ | ---------------------------------------------- |
| `summarize.py`     | Main script: read file → call Claude → print   |
| `input.txt`        | Sample long text (ISRO Chandrayaan-3 mission)  |
| `requirements.txt` | Pinned dependencies (`anthropic[vertex]`)      |
| `.venv/`           | Local virtual environment (gitignored)         |

## Notes

- Model is set via the `MODEL` constant in `summarize.py` (default: `claude-opus-4-8`).
  The model must be enabled in your Vertex project/region.
- On Vertex AI, model IDs use the bare first-party string (no `anthropic.` prefix).
- Project and region are read from `ANTHROPIC_VERTEX_PROJECT_ID` and
  `CLOUD_ML_REGION`; nothing is hardcoded or committed.

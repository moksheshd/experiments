"""Verify Google Drive / Docs / Sheets / Calendar API calls using desktop OAuth creds.

What it does:
  1. Lists files in your Drive.
  2. Reads a Google Doc  (Docs API).
  3. Reads a Google Sheet (Sheets API).
  4. Lists upcoming Calendar events (Calendar API).

Runs the installed-app OAuth flow (opens a browser once), caches the token in
token.json, then makes the calls above.

Usage:
  python verify_drive.py                     # auto-pick first Doc + Sheet found
  python verify_drive.py --doc DOC_ID --sheet SHEET_ID
"""

import argparse
import datetime
import os

from google.auth.transport.requests import Request
from google.oauth2.credentials import Credentials
from google_auth_oauthlib.flow import InstalledAppFlow
from googleapiclient.discovery import build

# Read-only scopes across the three APIs. Delete token.json if you change these.
SCOPES = [
    "https://www.googleapis.com/auth/drive.readonly",
    "https://www.googleapis.com/auth/documents.readonly",
    "https://www.googleapis.com/auth/spreadsheets.readonly",
    "https://www.googleapis.com/auth/calendar.readonly",
]

CLIENT_SECRETS = os.path.expanduser(
    "~/Downloads/client_secret_6444020331-fb8je361s4vtbsri9gqt54ebgoth2qld.apps.googleusercontent.com.json"
)
TOKEN_FILE = os.path.join(os.path.dirname(__file__), "token.json")

DOC_MIME = "application/vnd.google-apps.document"
SHEET_MIME = "application/vnd.google-apps.spreadsheet"


def get_credentials():
    creds = None
    if os.path.exists(TOKEN_FILE):
        creds = Credentials.from_authorized_user_file(TOKEN_FILE, SCOPES)

    if not creds or not creds.valid:
        if creds and creds.expired and creds.refresh_token:
            creds.refresh(Request())
        else:
            flow = InstalledAppFlow.from_client_secrets_file(CLIENT_SECRETS, SCOPES)
            creds = flow.run_local_server(port=0)
        with open(TOKEN_FILE, "w") as token:
            token.write(creds.to_json())

    return creds


def list_files(drive, page_size=20):
    print("=" * 60)
    print("1. LIST FILES")
    print("=" * 60)
    results = (
        drive.files()
        .list(
            pageSize=page_size,
            fields="files(id,name,mimeType,modifiedTime)",
            orderBy="modifiedTime desc",
        )
        .execute()
    )
    files = results.get("files", [])
    if not files:
        print("  (no files found)\n")
        return files

    for f in files:
        print(f"  - {f['name']}")
        print(f"      id={f['id']}  type={f['mimeType']}")
    print()
    return files


def read_doc(doc_id):
    print("=" * 60)
    print(f"2. READ GOOGLE DOC  ({doc_id})")
    print("=" * 60)
    docs = build("docs", "v1", credentials=CREDS)
    doc = docs.documents().get(documentId=doc_id).execute()

    print(f"  Title: {doc.get('title')}\n")

    # Flatten the document body into plain text.
    text_parts = []
    for element in doc.get("body", {}).get("content", []):
        paragraph = element.get("paragraph")
        if not paragraph:
            continue
        for run in paragraph.get("elements", []):
            content = run.get("textRun", {}).get("content")
            if content:
                text_parts.append(content)

    text = "".join(text_parts).strip()
    preview = text[:800]
    print("  --- content preview ---")
    print("\n".join("  " + line for line in preview.splitlines()) or "  (empty)")
    if len(text) > len(preview):
        print(f"  ... ({len(text)} chars total)")
    print()


def read_sheet(sheet_id):
    print("=" * 60)
    print(f"3. READ GOOGLE SHEET  ({sheet_id})")
    print("=" * 60)
    sheets = build("sheets", "v4", credentials=CREDS)

    meta = sheets.spreadsheets().get(spreadsheetId=sheet_id).execute()
    title = meta.get("properties", {}).get("title")
    tabs = [s["properties"]["title"] for s in meta.get("sheets", [])]
    print(f"  Title: {title}")
    print(f"  Tabs:  {', '.join(tabs)}\n")

    # Read the first tab's cells.
    first_tab = tabs[0]
    resp = (
        sheets.spreadsheets()
        .values()
        .get(spreadsheetId=sheet_id, range=first_tab)
        .execute()
    )
    rows = resp.get("values", [])
    print(f"  --- '{first_tab}' first rows ---")
    if not rows:
        print("  (empty)")
    for row in rows[:10]:
        print("  " + " | ".join(str(c) for c in row))
    if len(rows) > 10:
        print(f"  ... ({len(rows)} rows total)")
    print()


def read_calendar(max_results=10):
    print("=" * 60)
    print("4. LIST CALENDAR EVENTS")
    print("=" * 60)
    calendar = build("calendar", "v3", credentials=CREDS)

    # RFC3339 "now" in UTC; Calendar API wants the trailing Z.
    now = datetime.datetime.now(datetime.timezone.utc).isoformat()
    resp = (
        calendar.events()
        .list(
            calendarId="primary",
            timeMin=now,
            maxResults=max_results,
            singleEvents=True,
            orderBy="startTime",
        )
        .execute()
    )
    events = resp.get("items", [])
    print(f"  --- next {len(events)} upcoming events ---")
    if not events:
        print("  (no upcoming events)")
    for event in events:
        start = event.get("start", {})
        when = start.get("dateTime") or start.get("date") or "?"
        summary = event.get("summary", "(no title)")
        print(f"  - {when}  {summary}")
    print()


def pick(files, mime):
    for f in files:
        if f["mimeType"] == mime:
            return f["id"]
    return None


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--doc", help="Google Doc file ID to read")
    parser.add_argument("--sheet", help="Google Sheet file ID to read")
    args = parser.parse_args()

    global CREDS
    CREDS = get_credentials()
    drive = build("drive", "v3", credentials=CREDS)
    print("Authenticated.\n")

    files = list_files(drive)

    doc_id = args.doc or pick(files, DOC_MIME)
    if doc_id:
        read_doc(doc_id)
    else:
        print("2. READ GOOGLE DOC — skipped (no Doc found; pass --doc DOC_ID)\n")

    sheet_id = args.sheet or pick(files, SHEET_MIME)
    if sheet_id:
        read_sheet(sheet_id)
    else:
        print("3. READ GOOGLE SHEET — skipped (no Sheet found; pass --sheet SHEET_ID)\n")

    read_calendar()

    print("✅ Done.")


if __name__ == "__main__":
    main()

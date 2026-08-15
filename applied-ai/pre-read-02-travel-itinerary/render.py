"""Rich panels for the request/response learning view.

All rich usage lives here so main.py stays focused on the API call. Panels
render on stderr, keeping stdout reserved for the final JSON itinerary.
"""

import json

from rich.console import Console, Group
from rich.json import JSON
from rich.panel import Panel
from rich.text import Text

_console = Console(stderr=True)


def request(model: str, max_tokens: int, prompt: str, schema: dict) -> None:
    """Pretty-print the outgoing request: prompt text + the response schema."""
    header = Text.assemble(
        ("model:  ", "bold"), f"{model}\n",
        ("tokens: ", "bold"), f"max_output_tokens={max_tokens}\n\n",
        ("prompt:\n", "bold"), (prompt, "italic"),
        ("\n\nresponse_schema:", "bold"),
    )
    body = Group(header, JSON(json.dumps(schema)))
    _console.print(Panel(body, title="[bold cyan]→ Request[/]", border_style="cyan"))


def response(raw_json: str, usage, finish_reason) -> None:
    """Pretty-print the incoming response: token usage + the JSON body."""
    header = Text.assemble(
        ("finish_reason: ", "bold"), f"{finish_reason}\n",
        ("tokens in/out: ", "bold"),
        f"{usage.prompt_token_count}/{usage.candidates_token_count}\n\n",
        ("body:", "bold"),
    )
    body = Group(header, JSON(raw_json))
    _console.print(Panel(body, title="[bold green]← Response[/]", border_style="green"))

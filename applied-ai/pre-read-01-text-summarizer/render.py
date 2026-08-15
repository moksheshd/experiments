"""Rich panels for the request/response learning view.

All rich usage lives here so main.py stays focused on the API call. Panels
render on stderr, keeping stdout reserved for the summary itself.
"""

from rich.console import Console
from rich.panel import Panel
from rich.text import Text

_console = Console(stderr=True)


def request(model: str, max_tokens: int, system_prompt: str, text: str) -> None:
    """Pretty-print the outgoing request: system prompt + the input text."""
    preview = text if len(text) <= 800 else text[:800] + f"\n… [+{len(text) - 800:,} more chars]"
    body = Text.assemble(
        ("model:  ", "bold"), f"{model}\n",
        ("tokens: ", "bold"), f"max_output_tokens={max_tokens}\n\n",
        ("system_instruction:\n", "bold"), (system_prompt, "italic"),
        ("\n\ninput:\n", "bold"), preview,
    )
    _console.print(Panel(body, title="[bold cyan]→ Request[/]", border_style="cyan"))


def response(summary: str, usage) -> None:
    """Pretty-print the incoming response: token usage + the summary."""
    body = Text.assemble(
        ("tokens in/out: ", "bold"),
        f"{usage.prompt_token_count}/{usage.candidates_token_count}\n\n",
        ("summary:\n", "bold"), summary,
    )
    _console.print(Panel(body, title="[bold green]← Response[/]", border_style="green"))

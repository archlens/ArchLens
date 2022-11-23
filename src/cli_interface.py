import typer
from diagrams import Diagram

from src.core.bt_graph import BTGraph

DEFAULT_SETTINGS = {"diagram_name": "", "project": None}

app = typer.Typer()


@app.command()
def render(config_path: str, renderer: str):
    g = BTGraph()
    g.build_graph(config_path)

    diagram_name = g.DEFAULT_SETTINGS.get("diagram_name", "unknown")
    with Diagram(f"{diagram_name}-{renderer}", show=False):
        g.render_graph(renderer)


@app.command()
def validate(config_path: str):
    g = BTGraph()
    g.build_graph(config_path)
    if not g.validate_graph():
        exit(1)
    else:
        exit(0)


if __name__ == "__main__":
    app()

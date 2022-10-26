import typer
from diagrams import Diagram

import core as BT_core

DEFAULT_SETTINGS = {"diagram_name": "", "project": None}

app = typer.Typer()


@app.command()
def render(config_path: str):
    g = BT_core.BTGraph()
    g.build_graph(config_path)

    with Diagram(g.DEFAULT_SETTINGS.get("diagram_name", "unkown"), show=False):
        g.render_graph()


@app.command()
def validate(config_path: str):
    g = BT_core.BTGraph()
    g.build_graph(config_path)
    if not g.validate_graph():
        exit(1)
    else:
        exit(0)


if __name__ == "__main__":
    app()

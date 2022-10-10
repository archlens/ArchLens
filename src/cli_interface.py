import typer
from diagrams import Diagram

# from core import validate_graph
import os
import astroid
import core as BT_core

DEFAULT_SETTINGS = {"diagram_name": "", "project": None}

app = typer.Typer()


@app.command()
def render(config_path: str):
    g = BT_core.BTGraph()
    g.build_graph(config_path)

    with Diagram(DEFAULT_SETTINGS["diagram_name"], show=False):
        g.render_graph()


def _get_files_recursive(path: str) -> list[str]:
    file_list = []
    t = list(os.walk(path))
    for root, _, files in t:
        for file in files:
            file_list.append(os.path.join(root, file))

    file_list = [file for file in file_list if "__" not in file]

    return file_list


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

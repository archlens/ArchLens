import typer
from diagrams import Diagram
from core import validate_graph
import os

DEFAULT_SETTINGS = {"diagram_name": ""}

app = typer.Typer()


@app.command()
def render(config_path: str):
    os.environ["BT_DIAGRAM_RENDER"] = "true"
    source_code = _get_source_code(config_path)

    _compile_source_code(source_code)
    DEFAULT_SETTINGS.update(settings())

    with Diagram(DEFAULT_SETTINGS["diagram_name"], show=False):
        setup()

    del os.environ["BT_DIAGRAM_RENDER"]


@app.command()
def validate(config_path: str):
    source_code = _get_source_code(config_path)

    _compile_source_code(source_code)
    DEFAULT_SETTINGS.update(settings())

    graph = setup()
    is_valid = validate_graph(graph)
    print(is_valid)
    if not is_valid:
        exit(1)
    else:
        exit(0)


def setup():
    pass  # overridden by config file


def settings():
    pass  # overridden by config file


def _get_source_code(path):
    with open(path, "r") as file:
        code_str = file.read()
    return code_str


def _compile_source_code(source):
    code = compile(source, "config.py", "exec")
    exec(code, globals())


if __name__ == "__main__":
    app()

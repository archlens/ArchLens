import typer
from diagrams import Diagram
from core import validate_graph
import os
import astroid
import core as BT_core

DEFAULT_SETTINGS = {"diagram_name": "", "project": None}

app = typer.Typer()


@app.command()
def render(config_path: str):
    os.environ["BT_DIAGRAM_RENDER"] = "true"
    source_code = _get_source_code(config_path)

    _compile_source_code(source_code)
    DEFAULT_SETTINGS.update(settings())

    with Diagram(DEFAULT_SETTINGS["diagram_name"], show=False):
        try:
            nodes = setup()
            module_map = {node._ast.file: node for node in nodes if node._ast}

            project_ast = astroid.MANAGER.ast_from_module(
                DEFAULT_SETTINGS["project"]
            )
            file_list = _get_files_recursive(
                project_ast.file.replace("__init__.py", "")[:-1]
            )

            for file in file_list:
                if file not in module_map.keys():
                    node = BT_core.BTNode(label=file.split("/")[-1])
                    module_map[file] = node

            for file in file_list:
                file_ast = astroid.MANAGER.ast_from_file(file)
                imported_modules = BT_core.get_imported_modules(file_ast)
                module_map[file]._edge_to.extend(
                    [module_map[module.file] for module in imported_modules]
                )

            for node in module_map.values():
                unique_edges = list(
                    {edge.file: edge for edge in node._edge_to}.values()
                )
                node.diagram_node >> [
                    edge.diagram_node for edge in unique_edges
                ]
        except Exception as e:
            print(e)

    del os.environ["BT_DIAGRAM_RENDER"]


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

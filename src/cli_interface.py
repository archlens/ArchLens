import typer

from src.core.bt_graph import BTGraph

from src.plantuml.plantuml_file_creator import (
    plantuml_diagram_creator_entire_domain,
    plantuml_diagram_creator_sub_domains,
)

DEFAULT_SETTINGS = {"diagram_name": "", "project": None}


def render(config_path: str):
    g = BTGraph()
    g.build_graph(config_path)

    diagram_name = g.DEFAULT_SETTINGS.get("diagram_name", "unknown")

    plantuml_diagram_creator_entire_domain(
        g.root_module, diagram_name, save_location="./diagrams/"
    )

    # testing filtered view
    # views = ["test_project/tp_src/api", "test_project/tp_src/tp_core/sub_core"]
    # plantuml_diagram_creator_sub_domains(
    #     g.root_module, diagram_name, views, "./diagrams/"
    # )


def main():
    typer.run(render)


if __name__ == "__main__":
    typer.run(render)

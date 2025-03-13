import os

from src.views.view_entities import ViewPackage
import sys


def save_plant_uml(view_graph, view_name, config):
    plant_uml_str = _render_pu_graph(view_graph, view_name, config)
    project_name = config["name"]
    save_location = os.path.join(config["saveLocation"], f"{project_name}-{view_name}")
    _save_plantuml_str(save_location, plant_uml_str)


def save_plant_uml_diff(diff_graph, view_name, config):
    plant_uml_str = _render_pu_graph(diff_graph, view_name, config)
    project_name = config["name"]
    save_location = os.path.join(
        config["saveLocation"], f"{project_name}-diff-{view_name}"
    )
    _save_plantuml_str(save_location, plant_uml_str)


def _render_pu_graph(view_graph: list[ViewPackage], view_name, config):

    pu_package_string = "\n".join(
        [pu_package.render_package_pu() for pu_package in view_graph]
    )
    pu_dependency_string = "\n".join(
        [pu_package.render_dependency_pu() for pu_package in view_graph]
    )
    project_name = config.get("name", "")
    title = f"{project_name}-{view_name}"
    uml_str = f"""
@startuml
skinparam backgroundColor GhostWhite
title {title}
{pu_package_string}
{pu_dependency_string}
@enduml
        """

    if os.getenv("MT_DEBUG"):
        print(uml_str)
        print("Program Complete")
    return uml_str


def _save_plantuml_str(file_name: str, data: str):
    os.makedirs(os.path.dirname(file_name), exist_ok=True)
    with open(file_name, "w") as f:
        f.write(data)
    python_executable = sys.executable
    plantuml_server = os.getenv(
        "PLANTUML_SERVER_URL",
        "https://www.plantuml.com/plantuml/img/",
    )
    os.system(
        f"{python_executable} -m plantuml --server {plantuml_server}  {file_name}"
    )

    if os.path.exists(file_name):
        os.remove(file_name)

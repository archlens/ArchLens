import os

from src.views.view_entities import ViewPackage
import sys


def _get_common_prefix(names: list[str]) -> str:
    """Find the common prefix of all package names (dot-separated)."""
    if not names:
        return ""

    # Split all names by dots
    split_names = [name.split(".") for name in names]

    # Find common prefix parts
    common_parts = []
    for parts in zip(*split_names):
        if len(set(parts)) == 1:
            common_parts.append(parts[0])
        else:
            break

    # Return prefix (all but the last common part, to keep some context)
    if len(common_parts) > 0:
        return ".".join(common_parts)
    return ""


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
    # Find common prefix to strip from all names
    all_names = [pkg.name for pkg in view_graph]
    common_prefix = _get_common_prefix(all_names)

    # Strip prefix from names for cleaner display
    prefix_with_dot = common_prefix + "." if common_prefix else ""
    for pkg in view_graph:
        if pkg.name.startswith(prefix_with_dot):
            pkg.display_name = pkg.name[len(prefix_with_dot):]
        else:
            pkg.display_name = pkg.name

    pu_package_string = "\n".join(
        [pu_package.render_package_pu() for pu_package in view_graph]
    )
    pu_dependency_string = "\n".join(
        [pu_package.render_dependency_pu() for pu_package in view_graph]
    )
    project_name = config.get("name", "")

    # Include common prefix in title if present
    if common_prefix:
        title = f"{project_name}-{view_name}\\n<size:12>{common_prefix}.*</size>"
    else:
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

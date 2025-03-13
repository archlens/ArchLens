import os

from src.views.view_entities import ViewPackage
import json


def save_json(view_graph, view_name, config):
    json = _render_json_graph(view_graph, view_name, config)
    project_name = config["name"]
    save_location = os.path.join(config["saveLocation"], f"{project_name}-{view_name}")
    _save_json_file(save_location + ".json", json)


def save_json_diff(view_graph, view_name, config):
    json = _render_json_graph(view_graph, view_name, config)
    project_name = config["name"]
    save_location = os.path.join(
        config["saveLocation"], f"{project_name}-diff-{view_name}"
    )
    _save_json_file(save_location + ".json", json)


def _render_json_graph(view_graph: list[ViewPackage], view_name, config):
    json_packages = [json_package.render_package_json() for json_package in view_graph]

    json_dependencies = [
        relation
        for json_package in view_graph
        for relation in json_package.render_dependency_json()
    ]

    project_name = config.get("name", "")
    title = f"{project_name}-{view_name}"

    json_dict = {"title": title, "packages": json_packages, "edges": json_dependencies}

    if os.getenv("MT_DEBUG"):
        print(json.dumps(json_dict))
        print("Program Complete")
    return json_dict


def _save_json_file(save_location, json_dict):
    os.makedirs(os.path.dirname(save_location), exist_ok=True)
    with open(save_location, "w") as f:
        json.dump(json_dict, f, indent=4)

import typer
import json
import os
import requests
import jsonschema

from src.core.bt_graph import BTGraph
from src.plantuml.plantuml_file_creator import (
    plantuml_diagram_creator_sub_domains,
)

app = typer.Typer(add_completion=True)


@app.command()
def render(config_path: str = "mt_config.json"):
    config = read_config_file(config_path)
    g = BTGraph()
    g.build_graph(config)

    project_name = config.get("name")

    for view_name, views in config.get("views").items():
        formatted_views = [
            os.path.join(config.get("rootFolder"), view) for view in views["views"]
        ]
        plantuml_diagram_creator_sub_domains(
            g.root_module,
            f"{project_name}-{view_name}",
            formatted_views,
            views["ignoreModules"],
            save_location=config.get("saveLocation"),
        )


@app.command()
def create_config(config_path="./mt_config.json"):
    schema_url = "https://raw.githubusercontent.com/Perlten/MT-diagrams/master/config.template.json"
    os.makedirs(os.path.dirname(config_path), exist_ok=True)
    schema = requests.get(schema_url).json()
    with open(config_path, "w") as outfile:
        json.dump(schema, outfile, indent=4)


def read_config_file(config_path):
    schema_url = "https://raw.githubusercontent.com/Perlten/MT-diagrams/master/config.schema.json"
    config = None
    with open(config_path, "r") as f:
        config = json.load(f)

    schema = requests.get(schema_url).json()

    jsonschema.validate(instance=config, schema=schema)

    config["_config_path"] = os.path.dirname(config_path)
    return config


def main():
    app()


if __name__ == "__main__":
    main()

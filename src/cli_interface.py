import typer
import json
import os
import requests
import jsonschema

from src.core.bt_graph import BTGraph
from src.plantuml.plantuml_file_creator import (
    plantuml_diagram_creator_sub_domains,
)

def render(config_path: str):
    config = read_config_file(config_path)
    g = BTGraph()
    g.build_graph(config)

    project_name = config.get("name")



    for view_name, views in config.get("views").items():
        formatted_views = [
            os.path.join(config.get("rootFolder"), view) for view in views['view']
        ]
        x = 4
        plantuml_diagram_creator_sub_domains(
            g.root_module,
            f"{project_name}-{view_name}",
            formatted_views,
            views['ignoreModules'],
            save_location=config.get("saveLocation"),
        )

def read_config_file(config_path):
    schema_url = "https://raw.githubusercontent.com/Perlten/MT-diagrams/master/config.schema.json"
    config = None
    with open(config_path, "r") as f:
        config = json.load(f)

    schema = requests.get(schema_url).json()

    #jsonschema.validate(instance=config, schema=schema)

    config["_config_path"] = os.path.dirname(config_path)
    return config

def main():
    typer.run(render)



if __name__ == "__main__":
    typer.run(render)

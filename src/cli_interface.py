import typer
import json
import os
import requests
import jsonschema
import tempfile
import shutil
from pathlib import Path
from src.utils.path_manager_singleton import PathManagerSingleton
from src.utils.functions import verify_config_options
from src.utils.config_manager_singleton import ConfigManagerSingleton

from src.core.bt_graph import BTGraph

from src.plantuml.fetch_git import fetch_git_repo

from src.plantuml.plantuml_file_creator import (
    plantuml_diagram_creator_sub_domains,
)

app = typer.Typer(add_completion=True)


@app.command()
def render(config_path: str = "mt_config.json"):
    config = read_config_file(config_path)

    mt_path_manager = PathManagerSingleton()
    mt_path_manager.setup(config)

    g = BTGraph()
    g.build_graph(config)

    verify_config_options(config, g)

    project_name = config.get("name")

    for view_name, views in config.get("views").items():
        for view_name, views in config.get("views").items():
            formatted_views = []
            for view in views["packages"]:
                if type(view) == str:
                    formatted_views.append(
                        config.get("rootFolder") + f"/{view}"
                    )
                else:
                    view["packagePath"] = (
                        config.get("rootFolder") + "/" + view["packagePath"]
                    )
                    formatted_views.append(view)

        plantuml_diagram_creator_sub_domains(
            g.root_module,
            f"{project_name}-{view_name}",
            formatted_views,
            views["ignorePackages"],
            None,
            config.get("rootFolder"),
            views.get("usePackagePathAsLabel", True),
            save_location=config.get("saveLocation"),
        )


@app.command()
def render_diff(config_path: str = "mt_config.json"):
    with tempfile.TemporaryDirectory() as tmp_dir:
        print("Created temporary directory:", tmp_dir)
        config = read_config_file(config_path)

        fetch_git_repo(
            tmp_dir, config["github"]["url"], config["github"]["branch"]
        )

        shutil.copyfile(config_path, os.path.join(tmp_dir, "mt_config.json"))

        config_git = read_config_file(os.path.join(tmp_dir, "mt_config.json"))

        path_manager = PathManagerSingleton()
        path_manager.setup(config, config_git)

        g_git = BTGraph()
        g_git.build_graph(config_git)
        verify_config_options(config_git, g_git)

        g = BTGraph()
        g.build_graph(config)
        verify_config_options(config, g)

        project_name = config.get("name")

        for view_name, views in config.get("views").items():
            formatted_views = []
            for view in views["packages"]:
                if type(view) == str:
                    formatted_views.append(
                        config.get("rootFolder") + f"/{view}"
                    )
                else:
                    view["packagePath"] = (
                        config.get("rootFolder") + "/" + view["packagePath"]
                    )
                    formatted_views.append(view)
            plantuml_diagram_creator_sub_domains(
                g.root_module,
                f"{project_name}-{view_name}",
                formatted_views,
                views["ignorePackages"],
                g_git.root_module,
                config.get("rootFolder"),
                views.get("usePackagePathAsLabel", True),
                save_location=config.get("saveLocation"),
            )


@app.command()
def create_config(config_path="./mt_config.json"):
    schema_url = "https://raw.githubusercontent.com/Perlten/MT-diagrams/master/config.template.json"
    os.makedirs(os.path.dirname(config_path), exist_ok=True)
    schema = requests.get(schema_url).json()
    with open(config_path, "w") as outfile:
        json.dump(schema, outfile, indent=4)


@app.command()
def create_action():
    action_url = "https://raw.githubusercontent.com/Perlten/MT-diagrams/master/.github/workflows/pr-mt-diagrams.yml"
    action_path = Path(".github/workflows/pr-mt-diagrams.yml")
    typer.secho(f"Creating the action at {action_path}", fg="green")
    action_path.parent.mkdir(parents=True, exist_ok=True)
    action = requests.get(action_url).text

    with open(action_path, "w") as f:
        f.write(action)


def read_config_file(config_path):
    schema_url = "https://raw.githubusercontent.com/Perlten/MT-diagrams/master/config.schema.json"
    config = None
    with open(config_path, "r") as f:
        config = json.load(f)

    schema = requests.get(schema_url).json()

    if not os.getenv("MT_DEBUG"):
        jsonschema.validate(instance=config, schema=schema)

    config["_config_path"] = os.path.dirname(os.path.abspath(config_path))

    config_manager = ConfigManagerSingleton()
    config_manager.setup(config)

    return config


def main():
    app()


if __name__ == "__main__":
    main()

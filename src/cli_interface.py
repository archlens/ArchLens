import typer
import json
import os
import requests
import jsonschema
import tempfile
import shutil
from pathlib import Path
from src.utils.path_manager_singleton import PathManagerSingleton

# from src.utils.functions import verify_config_options
from src.utils.config_manager_singleton import ConfigManagerSingleton

from src.plantumlv2.pu_manager import render_pu, render_diff_pu

from src.core.bt_graph import BTGraph

from src.plantuml.fetch_git import fetch_git_repo

from astroid.manager import AstroidManager

import astroid

astroid.MANAGER = None

app = typer.Typer(add_completion=True)


@app.command()
def render(config_path: str = "archlens.json"):
    config = read_config_file(config_path)

    mt_path_manager = PathManagerSingleton()
    mt_path_manager.setup(config)

    am = _create_astroid()
    g = BTGraph(am)
    g.build_graph(config)

    render_pu(g, config)


def _create_astroid():
    am = AstroidManager()
    am.brain["astroid_cache"] = {}
    return am


@app.command()
def render_diff(config_path: str = "archlens.json"):
    with tempfile.TemporaryDirectory() as tmp_dir:
        print("Created temporary directory:", tmp_dir)
        config = read_config_file(config_path)

        fetch_git_repo(
            tmp_dir, config["github"]["url"], config["github"]["branch"]
        )

        shutil.copyfile(config_path, os.path.join(tmp_dir, "archlens.json"))

        config_git = read_config_file(os.path.join(tmp_dir, "archlens.json"))

        path_manager = PathManagerSingleton()
        path_manager.setup(config, config_git)

        local_am = _create_astroid()
        local_graph = BTGraph(local_am)
        local_graph.build_graph(config)
        # verify_config_options(config, g)

        remote_am = _create_astroid()
        remote_graph = BTGraph(remote_am)
        remote_graph.build_graph(config_git)
        # verify_config_options(config_git, g_git)

        render_diff_pu(local_graph, remote_graph, config)


@app.command()
def init(config_path="./archlens.json"):
    schema_url = "https://raw.githubusercontent.com/Perlten/MT-diagrams/master/config.template.json"
    os.makedirs(os.path.dirname(config_path), exist_ok=True)
    schema = requests.get(schema_url).json()
    schema["name"] = os.path.basename(os.getcwd())
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

from src.test_project.core import core
from src.test_project.api import api
from src.test_project.controller import controller
from src.test_project.core import core_extra

from src.core import BTNode
from src import test_project


def setup():
    api_node = BTNode(connected_code=api, label="api")
    core_node = BTNode(connected_code=core, label="core")
    core_extra_node = BTNode(connected_code=core_extra, label="core_extra")
    controller_node = BTNode(connected_code=controller, label="controller")
    third_party_api = BTNode(label="Awesome API")

    nodes = [
        api_node,
        core_node,
        controller_node,
        third_party_api,
        core_extra_node,
    ]

    api_node >> core_node
    core_node >> controller_node
    # core_node >> controller_node  # Comment for error
    # core_node >> core_extra_node  # Comment for error
    # api_node >> third_party_api
    return nodes


def settings():
    return {
        "diagram_name": "Test Project Diagram",
        "project": test_project,
    }

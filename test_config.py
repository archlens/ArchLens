from src.test_project.core import core
from src.test_project.api import api
from src.test_project.controller import controller

from src.core import BTNode


def setup():
    api_node = BTNode(connected_code=api, label="api")
    core_node = BTNode(connected_code=core, label="core")
    controller_node = BTNode(connected_code=controller, label="controller")
    third_party_api = BTNode(label="Awesome API")

    nodes = [api_node, core_node, controller_node, third_party_api]

    api_node >> core_node
    api_node >> controller_node
    core_node >> controller_node  # Comment for error
    api_node >> third_party_api

    return nodes


def settings():
    return {"diagram_name": "Test Project Diagram"}

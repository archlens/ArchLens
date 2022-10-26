from src.core.bt_node import BTNode
import test_project


def setup():
    api_node = BTNode(connected_code="api.api", label="hejkme ddig")
    core_node = BTNode(connected_code="tp_core.core", label="core")
    added_node = BTNode(label="Third party api")

    api_node == core_node
    api_node >> added_node

    return [api_node, core_node, added_node]


def settings():
    return {
        "diagram_name": "test project",
        "project": test_project,
    }

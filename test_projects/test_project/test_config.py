from src.core.bt_node import BTNode
import tp_src as test_project


def setup():
    api_node = BTNode(connected_code="tp_src.api.api", label="hejkme ddig")
    core_node = BTNode(connected_code="tp_src.tp_core.core", label="core")
    added_node = BTNode(label="Third party api")

    api_node == core_node
    api_node >> added_node

    return [api_node, core_node, added_node]


def settings():
    return {
        "diagram_name": "test project",
        "project": test_project,
    }

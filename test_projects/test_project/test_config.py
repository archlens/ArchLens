from src.core.bt_node import BTNode
import tp_src as test_project


def setup():
    api_node = BTNode(code_path="tp_src.api.api", label="api")
    core_node = BTNode(code_path="tp_src.tp_core.core", label="core")
    controller_node = BTNode(
        code_path="tp_src.controller.controller", label="controller"
    )
    added_node = BTNode(label="Third party api")

    api_node.whitelist(core_node)
    api_node.blacklist(controller_node)

    api_node >> added_node

    return [api_node, core_node, added_node]


def settings():
    return {
        "diagram_name": "test project",
        "project": test_project,
    }

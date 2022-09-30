from diagrams import Diagram
from diagrams import Node

from test_project.core import core
from test_project.api import api
from test_project.controller import controller

import astroid


def test_module_check():
    with Diagram("Test project diagram", show=False):
        api_node = BTNode(connected_code=api, label="api")
        core_node = BTNode(connected_code=core, label="core")
        controller_node = BTNode(connected_code=controller, label="controller")
        third_party_api = BTNode(label="Awesome API")

        nodes = [api_node, core_node, controller_node, third_party_api]

        api_node >> core_node
        api_node >> controller_node
        core_node >> controller_node
        api_node >> third_party_api

    for node in nodes:
        if not node.validate():
            print(f"error in node {node.label}")
            break

    print("done")


class BTNode:
    connected_code = None
    label: str = ""

    _edge_to: list["BTNode"] = []
    _ast = None
    diagram_node: Node = None

    def __init__(self, label: str, connected_code=None):
        self.connected_code = connected_code
        self.label = label
        self._ast = None
        if self.connected_code is not None:
            self._ast: astroid.Module = astroid.MANAGER.ast_from_module(
                connected_code
            )
        self._edge_to = []
        self.diagram_node = Node(label=label)

    def validate(self) -> bool:
        if self._ast is None:
            return True

        for sub_node in self._ast.body:
            if isinstance(sub_node, astroid.node_classes.ImportFrom):
                sub_node: astroid.node_classes.ImportFrom = sub_node
                if sub_node.modname not in [
                    edge._ast.name
                    for edge in self._edge_to
                    if edge._ast is not None
                ]:
                    return False
        return True

    def __rshift__(self, other: "BTNode"):
        self._edge_to.append(other)
        print(f"{self.label} -> {other.label}")
        self.diagram_node >> other.diagram_node


if __name__ == "__main__":
    # main()
    test_module_check()

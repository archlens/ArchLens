from graphviz import Digraph

from diagrams import Diagram
from diagrams.k8s.network import Ingress, Service
from diagrams import Node

from test_project.core import core
from test_project.api import api
from test_project.controller import controller

import astroid


def test_module_check():
    api_node = BTNode(connected_code=api, label="api")
    core_node = BTNode(connected_code=core, label="core")
    controller_node = BTNode(connected_code=controller, label="controller")

    nodes = [api_node, core_node, controller_node]

    api_node >> core_node
    api_node >> controller_node

    # core_node >> controller_node

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

    def __init__(self, connected_code, label: str):
        self.connected_code = connected_code
        self.label = label
        self._ast: astroid.Module = astroid.MANAGER.ast_from_module(
            connected_code
        )
        self._edge_to = []

    def validate(self) -> bool:
        for sub_node in self._ast.body:
            if isinstance(sub_node, astroid.node_classes.ImportFrom):
                sub_node: astroid.node_classes.ImportFrom = sub_node
                if sub_node.modname not in [
                    edge._ast.name for edge in self._edge_to
                ]:
                    return False
        return True

    def __rshift__(self, other: "BTNode"):
        self._edge_to.append(other)
        print(f"{self.label} -> {other.label}")


# def main():
#     print("start")
#     with Diagram("Grouped Workers", show=True):
#         api_node = BTNode(label="api", connected_code=api)
#         core_node = BTNode(label="core", connected_code=core)
#         test_node = BTNode(label="test")
#         graph = api_node >> core_node >> test_node
#
#         dot: Digraph = api_node._diagram.dot
#         dot_body = dot.body
#         edges = [edge for edge in dot_body if "->" in edge]
#         res = {}
#         for edge in edges:
#             l_edge = edge.split("->")[0]
#             r_edge = edge.split("->")[1]
#
#         dot_edges = dot.edge_attr
#         print("hej")
#

if __name__ == "__main__":
    # main()
    test_module_check()

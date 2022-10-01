from diagrams import Node
import os


import astroid


def validate_graph(nodes: list) -> bool:
    for node in nodes:
        if not node.validate():
            print(f"error in node {node.label}")
            return False
    return True


def _should_render():
    return os.getenv("BT_DIAGRAM_RENDER", "") == "true"


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
        if _should_render():
            self.diagram_node = Node(label=label)

    def validate(self) -> bool:
        if self._ast is None:
            return True

        for sub_node in self._ast.body:
            if isinstance(sub_node, astroid.node_classes.ImportFrom):
                sub_node: astroid.node_classes.ImportFrom = sub_node
                module_node = astroid.MANAGER.ast_from_module_name(
                    sub_node.modname
                )
                valid_edges = [
                    edge._ast.file
                    for edge in self._edge_to
                    if edge._ast is not None
                ]
                if module_node.file not in valid_edges:
                    return False
        return True

    def __rshift__(self, other: "BTNode"):
        self._edge_to.append(other)
        print(f"{self.label} -> {other.label}")
        if _should_render():
            self.diagram_node >> other.diagram_node

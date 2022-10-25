from hashlib import blake2b
import numbers
import sys
from diagrams import Node
import os


import astroid
from astroid.exceptions import AstroidImportError


class BTGraph:
    DEFAULT_SETTINGS = {"diagram_name": "", "project": None}
    graph: list["BTNode"] = []
    root_location: str = None

    def build_graph(self, config_path: str):
        root_location = "/".join(config_path.split("/")[:-1]) + "/"
        self.root_location = root_location

        source_code = self._get_source_code(config_path)

        self._compile_source_code(source_code, root_location)
        self.DEFAULT_SETTINGS.update(settings())

        nodes = setup()

        real_nodes = {node.ast.file: node for node in nodes if node.ast}
        extra_nodes = [node for node in nodes if not node.ast]

        file_list = self._get_files_recursive(root_location)

        for file in file_list:
            try:
                if file not in real_nodes.keys():
                    if not file.endswith(".py"):
                        continue
                    node = BTNode(label=file.split("/")[-1])
                    node.ast = astroid.MANAGER.ast_from_file(file)
                    real_nodes[file] = node
            except Exception as e:
                print(e)
                continue

        for file in file_list:
            try:
                if not file.endswith(".py"):
                    continue
                file_ast = astroid.MANAGER.ast_from_file(file)
                imported_modules = get_imported_modules(file_ast, root_location)

                real_nodes[file] >> [
                    real_nodes[module.file]
                    for module in imported_modules
                    if module.file is not None and module.file in real_nodes
                ]
            except Exception as e:
                print(e)
                continue

        self.graph = list(real_nodes.values())
        self.graph.extend(extra_nodes)

    def _get_files_recursive(self, path: str) -> list[str]:
        file_list = []
        t = list(os.walk(path))
        for root, _, files in t:
            for file in files:
                file_list.append(os.path.join(root, file))

        file_list = [file for file in file_list]

        return file_list

    def _get_source_code(self, path):
        with open(path, "r") as file:
            code_str = file.read()
        return code_str

    def _compile_source_code(self, source, root_location: str):
        sys.path.append(root_location)
        code = compile(source, "config.py", "exec")
        exec(code, globals())

    def validate_graph(self) -> bool:
        for node in self.graph:
            if not node.validate():
                print(f"error in node {node.label}")
                return False
        return True

    def render_graph(self):
        node_map = {}
        for node in self.graph:
            f = "".join(node.file.rsplit(self.root_location))
            n = Node(label=f)
            node_map[node.uid] = n

        for node in self.graph:
            edges = node.edge_to
            node_map[node.uid] >> [node_map[edge.uid] for edge in edges]


def setup():
    pass  # overridden by config file


def settings():
    pass  # overridden by config file


def get_imported_modules(ast: astroid.Module, root_location: str):
    imported_modules = []
    for sub_node in ast.body:
        try:
            if isinstance(sub_node, astroid.node_classes.ImportFrom):
                sub_node: astroid.node_classes.ImportFrom = sub_node

                module_node = astroid.MANAGER.ast_from_module_name(
                    sub_node.modname,
                    context_file=root_location,
                )
                imported_modules.append(module_node)

            if isinstance(sub_node, astroid.node_classes.Import):
                pass  # TODO!!

        except AstroidImportError:
            continue

    return imported_modules


class BTNode:
    connected_code = None
    label: str = ""
    policies: list["BTPolicy"] = None

    edge_to: list["BTNode"] = None
    ast = None
    uid: str = None

    def __init__(self, label: str, connected_code=None):
        print(f"create {label}")
        self.connected_code = connected_code
        self.label = label
        self.ast = None
        if self.connected_code is not None:
            self.ast: astroid.Module = astroid.MANAGER.ast_from_module(connected_code)
            self.uid = self.ast.file
        else:
            self.uid = label  # TODO Make this actual uid
        self.edge_to = []
        self.policies = []

    @property
    def file(self):
        if self.ast:
            return self.ast.file
        return ""

    def validate(self) -> bool:
        for policy in self.policies:
            if not policy.validate():
                return False
        return True

    def __rshift__(self, other):
        if isinstance(other, list):
            existing_edges = set(
                [edge.file for edge in self.edge_to if edge.file != ""]
            )
            new_node_list = filter(lambda e: e.file not in existing_edges, other)
            self.edge_to.extend([node for node in new_node_list])
        else:
            edges = set([edge.file for edge in self.edge_to])
            if other.file in edges:
                return

            self.edge_to.append(other)

    def __ne__(self, other):
        policy = BlacklistPolicy(self.edge_to, other)
        self.policies.append(policy)

    def __eq__(self, other):
        policy = WhitelistPolicy(self.edge_to, other)
        self.policies.append(policy)


class BTPolicy:
    def validate() -> bool:
        raise Exception("validate not implemented")


class BlacklistPolicy(BTPolicy):
    edges: list[BTNode] = None
    blacklisted_node: BTNode = None

    def __init__(self, edges, blacklisted_node) -> None:
        super().__init__()
        self.edges = edges
        self.blacklisted_node = blacklisted_node

    def validate(self) -> bool:
        for edge in self.edges:
            if edge.file == self.blacklisted_node.file:
                return False
        return True


class WhitelistPolicy(BTPolicy):
    edges: list[BTNode] = None
    whitelisted_node: BTNode = None

    def __init__(self, edges, whitelisted_node) -> None:
        super().__init__()
        self.edges = edges
        self.whitelisted_node = whitelisted_node

    def validate(self) -> bool:
        for edge in self.edges:
            if edge.file == self.whitelisted_node.file:
                return True
        return False

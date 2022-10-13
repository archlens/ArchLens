from hashlib import blake2b
import numbers
from diagrams import Node
import os


import astroid


class BTGraph:
    DEFAULT_SETTINGS = {"diagram_name": "", "project": None}
    graph: list["BTNode"] = []

    def build_graph(self, config_path: str):
        source_code = self._get_source_code(config_path)

        self._compile_source_code(source_code)
        self.DEFAULT_SETTINGS.update(settings())

        nodes = setup()

        real_nodes = {node.ast.file: node for node in nodes if node.ast}
        extra_nodes = [node for node in nodes if not node.ast]

        project_ast = astroid.MANAGER.ast_from_module(self.DEFAULT_SETTINGS["project"])
        file_list = self._get_files_recursive(
            project_ast.file.replace("__init__.py", "")[:-1]
        )

        for file in file_list:
            if file not in real_nodes.keys():
                if not file.endswith(".py"):
                    continue
                node = BTNode(label=file.split("/")[-1])
                node.ast = astroid.MANAGER.ast_from_file(file)
                real_nodes[file] = node

        for file in file_list:
            file_ast = astroid.MANAGER.ast_from_file(file)
            imported_modules = get_imported_modules(file_ast)
            real_nodes[file] >> [real_nodes[module.file] for module in imported_modules]

        self.graph = list(real_nodes.values())
        self.graph.extend(extra_nodes)

    def _get_files_recursive(self, path: str) -> list[str]:
        file_list = []
        t = list(os.walk(path))
        for root, _, files in t:
            for file in files:
                file_list.append(os.path.join(root, file))

        file_list = [file for file in file_list if "__" not in file]

        return file_list

    def _get_source_code(self, path):
        with open(path, "r") as file:
            code_str = file.read()
        return code_str

    def _compile_source_code(self, source):
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
            n = Node(label=node.label)
            node_map[node.uid] = n

        for node in self.graph:
            edges = node.edge_to
            node_map[node.uid] >> [node_map[edge.uid] for edge in edges]


def setup():
    pass  # overridden by config file


def settings():
    pass  # overridden by config file


def get_imported_modules(ast: astroid.Module):
    imported_modules = []
    for sub_node in ast.body:
        if isinstance(sub_node, astroid.node_classes.ImportFrom):
            sub_node: astroid.node_classes.ImportFrom = sub_node

            module_node = astroid.MANAGER.ast_from_module_name(sub_node.modname)

            imported_modules.append(module_node)

        if isinstance(sub_node, astroid.node_classes.Import):
            pass  # TODO!!

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

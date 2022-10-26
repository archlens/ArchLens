import astroid
import sys
import os

from src.core.bt_node import BTNode, get_imported_modules
import diagrams

from src.core.policies import BlacklistPolicy, WhitelistPolicy


class BTGraph:
    DEFAULT_SETTINGS = {"diagram_name": "", "project": None}
    graph: list[BTNode] = []
    target_project_location: str = None

    def build_graph(self, config_path: str):
        target_project_location = "/".join(config_path.split("/")[:-1]) + "/"
        self.target_project_location = target_project_location

        source_code = self._get_source_code(config_path)

        self._compile_source_code(source_code)
        self.DEFAULT_SETTINGS.update(settings())

        nodes = setup()

        real_nodes = {node.ast.file: node for node in nodes if node.ast}
        extra_nodes = [node for node in nodes if not node.ast]

        file_list = self._get_files_recursive(self.target_project_location)

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
                imported_modules = get_imported_modules(
                    file_ast, self.target_project_location
                )

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

    def _compile_source_code(self, source):
        sys.path.append(self.target_project_location)
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
        for bt_node in self.graph:
            label = "".join(bt_node.file.rsplit(self.target_project_location))
            n = diagrams.Node(label=label)
            node_map[bt_node.uid] = n

        for bt_node in self.graph:
            edges = bt_node.edge_to
            diagram_node = node_map[bt_node.uid]
            white_listed_nodes = [
                policy.whitelisted_node
                for policy in bt_node.policies
                if isinstance(policy, WhitelistPolicy)
            ]

            black_listed_nodes = [
                policy.blacklisted_node
                for policy in bt_node.policies
                if isinstance(policy, BlacklistPolicy)
            ]

            for edge in edges:
                edge_node = node_map[edge.uid]
                if edge.uid in [white_node.uid for white_node in white_listed_nodes]:
                    diagram_node >> diagrams.Edge(color="blue") >> edge_node
                elif edge.uid in [black_node.uid for black_node in black_listed_nodes]:
                    diagram_node >> diagrams.Edge(color="red") >> edge_node
                else:
                    diagram_node >> edge_node


def setup():
    pass  # overridden by config file


def settings():
    pass  # overridden by config file

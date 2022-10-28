import astroid
import sys
import os

from src.core.bt_file import BTFile, get_imported_modules
import diagrams
from src.core.bt_module import BTModule

from src.core.policies import BlacklistPolicy, WhitelistPolicy


class BTGraph:
    DEFAULT_SETTINGS = {"diagram_name": "", "project": None}
    graph: list[BTFile] = []
    modules: list[BTModule] = []
    target_project_location: str = None

    def build_graph(self, config_path: str):
        target_project_location = "/".join(config_path.split("/")[:-1]) + "/"
        self.target_project_location = target_project_location

        source_code = self._get_source_code(config_path)

        self._compile_source_code(source_code)
        self.DEFAULT_SETTINGS.update(settings())

        nodes = setup()

        bt_file_list = {node.ast.file: node for node in nodes if node.ast}
        extra_nodes = [node for node in nodes if not node.ast]

        bt_module_list: list[BTModule] = []

        file_list = self._get_files_recursive(self.target_project_location)

        # Create modules
        for file in file_list:
            try:
                if not file.endswith("__init__.py"):
                    continue
                bt_module = BTModule(file)
                bt_module_list.append(bt_module)
            except Exception as e:
                print(e)
                continue

        for module in bt_module_list:
            for parent_module in bt_module_list:
                if module == parent_module:
                    continue
                t = "/".join(module.path.split("/")[:-1])
                if parent_module.path == t:
                    parent_module.child_module.append(module)
                    module.parent_module = parent_module

        # Create all bt files
        for file in file_list:
            try:
                if file not in bt_file_list.keys():
                    if not file.endswith(".py") or file.endswith("__init__.py"):
                        continue
                    node = BTFile(label=file.split("/")[-1])
                    node.ast = astroid.MANAGER.ast_from_file(file)
                    bt_module = next(
                        (x for x in bt_module_list if x.path == node.module_path), None
                    )
                    bt_file_list[file] = node
                    if bt_module:
                        bt_module.file_list.append(node)
            except Exception as e:
                print(e)
                continue

        for file in file_list:
            try:
                if file not in bt_file_list:
                    continue
                file_ast = astroid.MANAGER.ast_from_file(file)
                imported_modules = get_imported_modules(
                    file_ast, self.target_project_location
                )

                bt_file_list[file] >> [
                    bt_file_list[module.file]
                    for module in imported_modules
                    if module.file is not None and module.file in bt_file_list
                ]
            except Exception as e:
                print(e)
                continue

        self.graph = list(bt_file_list.values())
        self.graph.extend(extra_nodes)
        self.modules = bt_module_list

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

        def create_nodes(module: BTModule):
            with diagrams.Cluster(label=module.name):
                for bt_node in module.file_list:
                    n = diagrams.Node(label=bt_node.label)
                    node_map[bt_node.uid] = n
                for child_module in module.child_module:
                    create_nodes(child_module)

        root_modules = list(filter(lambda e: not e.parent_module, self.modules))
        for module in root_modules:
            create_nodes(module)

        def render_module(module: BTModule):
            for bt_node in module.file_list:
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
                    if edge.uid not in node_map:
                        continue
                    edge_node = node_map[edge.uid]
                    if edge.uid in [
                        white_node.uid for white_node in white_listed_nodes
                    ]:
                        diagram_node >> diagrams.Edge(color="blue") >> edge_node
                    elif edge.uid in [
                        black_node.uid for black_node in black_listed_nodes
                    ]:
                        diagram_node >> diagrams.Edge(color="red") >> edge_node
                    else:
                        diagram_node >> edge_node

            for child_module in module.child_module:
                render_module(child_module)

        root_modules = list(filter(lambda e: not e.parent_module, self.modules))

        for module in root_modules:
            render_module(module)


def setup():
    pass  # overridden by config file


def settings():
    pass  # overridden by config file

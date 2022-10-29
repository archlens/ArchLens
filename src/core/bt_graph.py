import astroid
import sys
import os

from src.core.bt_file import BTFile, get_imported_modules
import diagrams
from diagrams.programming.language import Python as pythonNode
from src.core.bt_module import BTModule

from src.core.policies import BlacklistPolicy, WhitelistPolicy


class BTGraph:
    DEFAULT_SETTINGS = {"diagram_name": "", "project": None}
    root_module_location: str = None
    target_project_base_location: str = None
    root_module = None

    def build_graph(self, config_path: str):
        source_code = self._get_source_code(config_path)
        self._compile_source_code(source_code, os.path.dirname(config_path))
        self.DEFAULT_SETTINGS.update(settings())

        self.root_module_location = os.path.dirname(
            self.DEFAULT_SETTINGS["project"].__file__
        )
        self.target_project_base_location = os.path.dirname(config_path)

        nodes = setup()  # TODO !

        bt_module_list: list[BTModule] = []

        file_list = self._get_files_recursive(self.root_module_location)

        # Create modules
        for file in file_list:
            try:
                if not file.endswith("__init__.py"):
                    continue
                bt_module = BTModule(file)
                bt_module.add_files()
                bt_module_list.append(bt_module)
            except Exception as e:
                print(e)
                continue

        for module in bt_module_list:
            for parent_module in bt_module_list:
                if module == parent_module:
                    continue
                if parent_module.path == "/".join(module.path.split("/")[:-1]):
                    parent_module.child_module.append(module)
                    module.parent_module = parent_module

        self.root_module = next(
            filter(lambda e: e.parent_module is None, bt_module_list)
        )

        # Set BTFiles dependencies
        btf_map = self.get_all_bf_files_map()
        for bt_file in btf_map.values():
            imported_modules = get_imported_modules(
                bt_file.ast, self.target_project_base_location
            )
            bt_file >> [
                btf_map[module.file]
                for module in imported_modules
                if module.file in btf_map
            ]

    def get_all_bf_files_map(self) -> dict[str, BTFile]:
        def get_bt_files(module: BTModule) -> list[BTFile]:
            bt_files: dict[str, BTFile] = {}

            bt_files.update({btf.file: btf for btf in module.file_list})

            for child_module in module.child_module:
                bt_files.update(get_bt_files(child_module))

            return bt_files

        return get_bt_files(self.root_module)

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

    def _compile_source_code(self, source, config_folder):
        sys.path.append(config_folder)
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

        create_nodes(self.root_module)

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

        render_module(self.root_module)


def setup():
    pass  # overridden by config file


def settings():
    pass  # overridden by config file

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

        try:

            os.environ["EDGE_AS_RULE"] = "true"
            nodes = setup()
            del os.environ["EDGE_AS_RULE"]
            real_nodes = {node._ast.file: node for node in nodes if node._ast}
            extra_nodes = [node for node in nodes if not node._ast]

            project_ast = astroid.MANAGER.ast_from_module(
                self.DEFAULT_SETTINGS["project"]
            )
            file_list = self._get_files_recursive(
                project_ast.file.replace("__init__.py", "")[:-1]
            )

            for file in file_list:
                if file not in real_nodes.keys():
                    node = BTNode(label=file.split("/")[-1])
                    node._ast = astroid.MANAGER.ast_from_file(file)
                    real_nodes[file] = node

            for file in file_list:
                file_ast = astroid.MANAGER.ast_from_file(file)
                imported_modules = get_imported_modules(file_ast)
                real_nodes[file] >> [
                    real_nodes[module.file] for module in imported_modules
                ]

            self.graph = list(real_nodes.values())
            self.graph.extend(extra_nodes)

        except Exception as e:
            print(e)

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
            edges = node._edge_to
            node_map[node.uid] >> [node_map[edge.node.uid] for edge in edges]


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

    return imported_modules


class BTNode:
    connected_code = None
    label: str = ""

    _edge_to: list["BTEdge"] = None
    _ast = None
    uid: str = None

    def __init__(self, label: str, connected_code=None):
        print(f"create {label}")
        self.connected_code = connected_code
        self.label = label
        self._ast = None
        if self.connected_code is not None:
            self._ast: astroid.Module = astroid.MANAGER.ast_from_module(connected_code)
            self.uid = self._ast.file
        else:
            self.uid = label  # TODO Make this actual uid
        self._edge_to = []

    @property
    def file(self):
        if self._ast:
            return self._ast.file
        return ""

    def validate(self) -> bool:
        if self._ast is None:
            return True

        for sub_node in self._ast.body:
            if isinstance(sub_node, astroid.node_classes.ImportFrom):
                sub_node: astroid.node_classes.ImportFrom = sub_node
                module_node = astroid.MANAGER.ast_from_module_name(sub_node.modname)
                valid_edges = [
                    edge._ast.file for edge in self._edge_to if edge._ast is not None
                ]
                if module_node.file not in valid_edges:
                    return False
        return True

    def __rshift__(self, other, rule=False):
        if os.getenv("EDGE_AS_RULE", "") == "true":
            rule = True
        if isinstance(other, list):
            existing_edges = set(
                [edge.node.file for edge in self._edge_to if edge.node.file != ""]
            )
            new_node_list = filter(lambda e: e.file not in existing_edges, other)
            self._edge_to.extend([BTEdge(node, rule) for node in new_node_list])
        else:
            edges = set([edge.file for edge in self._edge_to])
            if other.file in edges:
                return

            self._edge_to.append(BTEdge(other, rule))


class BTEdge:
    node: BTNode
    is_rule: bool

    def __init__(self, node: BTNode, is_rule=False) -> None:
        self.node = node
        self.is_rule = is_rule

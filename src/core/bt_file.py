import astroid
from astroid.manager import AstroidManager

from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from src.core.bt_module import BTModule


class BTFile:
    label: str = ""
    edge_to: list["BTFile"] = None
    ast = None
    module: "BTModule" = None
    am: AstroidManager

    def __init__(
        self, label: str, module, am: AstroidManager, code_path: str = None
    ):
        self.label = label
        self.ast = None
        self.am = am

        if code_path is not None:
            self.ast: astroid.Module = self.am.ast_from_module_name(code_path)

        self.edge_to = []
        self.module = module

    @property
    def file(self):
        if self.ast:
            return self.ast.file
        return ""

    @property
    def uid(self):
        if self.ast:
            return self.ast.file
        else:
            return self.label

    @property
    def module_path(self) -> str:
        if not self.ast:
            return None
        return "/".join(self.file.split("/")[:-1])

    def __rshift__(self, other):
        if isinstance(other, list):
            existing_edges = set(
                [edge.file for edge in self.edge_to if edge.file != ""]
            )
            new_node_list = filter(
                lambda e: e.file not in existing_edges, other
            )
            self.edge_to.extend([node for node in new_node_list])
        else:
            edges = set([edge.file for edge in self.edge_to])
            if other.file in edges:
                return

            self.edge_to.append(other)


def get_imported_modules(
    ast: astroid.Module, root_location: str, am: AstroidManager
) -> list:
    imported_modules = []
    for sub_node in ast.body:
        try:
            if isinstance(sub_node, astroid.node_classes.ImportFrom):
                sub_node: astroid.node_classes.ImportFrom = sub_node

                module_node = am.ast_from_module_name(
                    sub_node.modname,
                    context_file=root_location,
                )
                imported_modules.append(module_node)

            elif isinstance(sub_node, astroid.node_classes.Import):
                for name, _ in sub_node.names:
                    try:
                        module_node = am.ast_from_module_name(
                            name,
                            context_file=root_location,
                        )
                        imported_modules.append(module_node)
                    except Exception:
                        continue
            elif hasattr(sub_node, "body"):
                imported_modules.extend(
                    get_imported_modules(sub_node, root_location, am)
                )

        except astroid.AstroidImportError:
            continue

    return imported_modules

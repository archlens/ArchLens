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

    def __init__(self, label: str, module, am: AstroidManager, code_path: str = None):
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

    @property
    def path(self) -> str:
        """Returns the full file path (used for ViewPackage compatibility)"""
        if not self.ast:
            return None
        return self.ast.file

    @property
    def depth(self) -> int:
        """Depth is module depth + 1 (file is one level deeper than its module)"""
        if self.module:
            return self.module.depth + 1
        return 0

    @property
    def parent_module(self):
        """Alias for module (ViewPackage compatibility)"""
        return self.module

    @property
    def child_module(self) -> list:
        """Files don't have children"""
        return []

    @property
    def name(self) -> str:
        """Return file name without .py extension"""
        return self.label.replace(".py", "")

    def get_submodules_recursive(self) -> list:
        """Files don't have submodules"""
        return []

    def get_module_dependencies(self) -> set["BTFile"]:
        """Get all files this file depends on"""
        return set(self.edge_to)

    def get_file_dependencies(self) -> set["BTFile"]:
        """Get all files this file depends on (same as get_module_dependencies for files)"""
        return set(self.edge_to)

    def get_dependency_count(self, other: "BTFile") -> int:
        """Count how many times this file imports from another file"""
        if isinstance(other, BTFile):
            return 1 if other in self.edge_to else 0
        # If other is a BTModule, count edges to files in that module
        count = len([e for e in self.edge_to if e.module == other])
        return count

    def get_file_level_relations(self, target) -> list:
        """Get file-level relations to target (file or module)"""
        if isinstance(target, BTFile):
            if target in self.edge_to:
                return [(self, target)]
            return []
        # If target is a BTModule, get relations to files in that module
        return [(self, e) for e in self.edge_to if e.module == target]

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


def _resolve_relative_import(modname: str, level: int, current_file: str) -> str:
    """
    Resolve a relative import to an absolute module name.

    Args:
        modname: The module name from the import (e.g., "prompts" for "from .prompts import ...")
        level: Number of dots (1 for ".", 2 for "..", etc.)
        current_file: Path to the file containing the import

    Returns:
        The absolute module name
    """
    import os

    # Get the directory of the current file
    current_dir = os.path.dirname(current_file)

    # Go up 'level' directories (level=1 means current package, level=2 means parent, etc.)
    for _ in range(level - 1):
        current_dir = os.path.dirname(current_dir)

    # Convert path to module format
    # Find the package name from the directory
    package_parts = []
    temp_dir = current_dir
    while os.path.exists(os.path.join(temp_dir, "__init__.py")):
        package_parts.insert(0, os.path.basename(temp_dir))
        parent = os.path.dirname(temp_dir)
        if parent == temp_dir:  # Reached filesystem root
            break
        temp_dir = parent

    base_module = ".".join(package_parts)

    if modname:
        return f"{base_module}.{modname}" if base_module else modname
    return base_module


def get_imported_modules(
    ast: astroid.Module, root_location: str, am: AstroidManager
) -> list:
    imported_modules = []

    # Get the file path from the root module (works even when called with nested nodes)
    root_module = ast.root() if hasattr(ast, 'root') else ast
    current_file = root_module.file if hasattr(root_module, 'file') else None

    # Use nodes_of_class to find ALL imports in the entire AST tree,
    # including those inside functions and classes
    for sub_node in ast.nodes_of_class((astroid.node_classes.ImportFrom, astroid.node_classes.Import)):
        try:
            if isinstance(sub_node, astroid.node_classes.ImportFrom):
                modname = sub_node.modname
                level = sub_node.level if sub_node.level else 0

                # Handle relative imports
                if level > 0 and current_file:
                    modname = _resolve_relative_import(modname or "", level, current_file)

                if modname:
                    module_node = am.ast_from_module_name(
                        modname,
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

        except astroid.AstroidImportError:
            continue
        except Exception:
            continue

    return imported_modules

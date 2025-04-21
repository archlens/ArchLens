import astroid
import os

from src.core.bt_file import BTFile
from astroid.manager import AstroidManager


class BTModule:
    parent_module: "BTModule" = None
    child_module: list["BTModule"] = None

    name_if_duplicate_exists = None

    file_list: list["BTFile"] = None

    ast: astroid.Module = None
    am: AstroidManager = None

    def __init__(self, file_path: str, am: AstroidManager) -> None:
        self.ast = am.ast_from_file(file_path)
        self.child_module = []
        self.file_list = []
        self.am = am
        print(f"analyzing {self.path}")

    @property
    def depth(self):
        parents = self.get_parent_module_recursive()
        return len(parents)

    @property
    def name(self):
        return self.ast.name.split(".")[-1]

    @property
    def path(self):
        return os.path.dirname(self.ast.file)

    def add_files(self):
        py_files = [
            element for element in os.listdir(self.path)
            if element.endswith(".py") and element != "testing.py"
        ]

        self.file_list = [None] * len(py_files)
        i = 0

        for file in py_files:
            file_path = os.path.join(self.path, file)

            bt_file = BTFile(label=file.split("/")[-1], module=self, am=self.am)
            bt_file.ast = self.am.ast_from_file(file_path)
            self.file_list[i] = bt_file
            i += 1

    def get_files_recursive(self) -> list[BTFile]:
        temp_file_list = self.file_list.copy()
        for child_module in self.child_module:
            temp_file_list.extend(child_module.get_files_recursive())
        return temp_file_list

    def get_submodules_recursive(self) -> list["BTModule"]:
        submodule_list = self.child_module.copy()
        for submodule in self.child_module:
            submodule_list.extend(submodule.get_submodules_recursive())
        return submodule_list

    def get_parent_module_recursive(self):
        if self.parent_module is None:
            return []
        parent_module_list = [self.parent_module]
        parent_module_list.extend(self.parent_module.get_parent_module_recursive())
        return parent_module_list

    def get_module_dependencies(self) -> set["BTModule"]:
        dependencies = set()
        for child in self.file_list:
            dependencies.update(map(lambda e: e.module, child.edge_to))
        return dependencies

    def get_dependency_count(self, other: "BTModule"):
        file_dependencies = other.file_list
        files = [edge for element in self.file_list for edge in element.edge_to]
        count = len([element for element in files if element in file_dependencies])
        return count

    def get_file_level_relations(self, target_module: "BTModule"):
        """
        files contained in this object that depend on :param other:
        :return:
        """

        # relation is a tuple (file, file)
        all_file_relations = [
            (origin_file, target_file)
            for origin_file in self.file_list
            for target_file in origin_file.edge_to
        ]
        relations = [
            relation
            for relation in all_file_relations
            if relation[1] in target_module.file_list
        ]
        return relations

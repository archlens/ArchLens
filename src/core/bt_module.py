import astroid
import os

from src.core.bt_file import BTFile


class BTModule:
    parent_module: "BTModule" = None
    child_module: list["BTModule"] = None
    
    name_if_duplicate_exists = None

    file_list: list["BTFile"] = None

    ast: astroid.Module = None

    def __init__(self, file_path: str) -> None:
        self.ast = astroid.MANAGER.ast_from_file(file_path)
        self.child_module = []
        self.file_list = []

    @property
    def name(self):
        return self.ast.name.split(".")[-1]

    @property
    def path(self):
        return "/".join(self.ast.file.split("/")[:-1])

    def add_files(self):
        files = [
            element
            for element in os.listdir(self.path)
            if element.endswith(".py") and "__" not in element
        ]

        for file in files:
            bt_file = BTFile(label=file.split("/")[-1], module=self)
            bt_file.ast = astroid.MANAGER.ast_from_file(os.path.join(self.path, file))
            self.file_list.append(bt_file)

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

    def get_module_dependencies(self):
        dependencies = set()
        for child in self.file_list:
            dependencies.update(map(lambda e: e.module, child.edge_to))
        return dependencies

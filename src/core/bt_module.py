import astroid
import os

from src.core.bt_file import BTFile
from src.core.policies.ModulePolicies import BTModulePolicy, ModulePolicyCantDepend


class BTModule:
    parent_module: "BTModule" = None
    child_module: list["BTModule"] = None

    file_list: list["BTFile"] = None

    ast: astroid.Module = None
    policies: list[BTModulePolicy] = None

    def __init__(self, file_path: str) -> None:
        self.ast = astroid.MANAGER.ast_from_file(file_path)
        self.child_module = []
        self.file_list = []
        self.policies = []

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
            bt_file = BTFile(label=file.split("/")[-1])
            bt_file.ast = astroid.MANAGER.ast_from_file(os.path.join(self.path, file))
            self.file_list.append(bt_file)

    def get_all_files_recursive(self) -> list[BTFile]:
        def inner_function(module: BTModule):
            temp_file_list = module.file_list.copy()
            for child_module in module.child_module:
                temp_file_list.extend(inner_function(child_module))
            return temp_file_list

        return inner_function(self)

    def get_submodules_recursive(self) -> list["BTModule"]:
        submodule_list = self.child_module.copy()
        for submodule in self.child_module:
            submodule_list.extend(submodule.get_submodules_recursive())
        return submodule_list

    def validate(self) -> bool:
        for policy in self.policies:
            if not policy.validate(self):
                return False
        return True

    def cant_depend(self, other: "BTModule"):
        self.policies.append(ModulePolicyCantDepend(other))

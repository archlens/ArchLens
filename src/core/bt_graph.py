import astroid
import sys
import os

from src.core.bt_file import BTFile, get_imported_modules
from src.core.bt_module import BTModule
from astroid.manager import AstroidManager


class BTGraph:
    DEFAULT_SETTINGS = {"diagram_name": "", "project": None}
    root_module_location: str = None
    target_project_base_location: str = None
    root_module = None
    base_module = None
    am: AstroidManager = None

    def __init__(self, am: AstroidManager) -> None:
        self.am = am

    def build_graph(self, config: dict):
        config_path = config.get("_config_path")
        self.root_module_location = os.path.join(
            config_path, config.get("rootFolder")
        )
        self.target_project_base_location = config_path

        sys.path.insert(0, config_path)
        sys.path.insert(1, self.root_module_location)

        bt_module_list: list[BTModule] = []

        # Read all the python files within the project
        file_list = self._get_files_recursive(self.root_module_location)

        # Create modules and add them to module list (No relations between them yet)
        for file in file_list:
            try:
                if not file.endswith("__init__.py"):
                    continue
                bt_module = BTModule(file, self.am)
                bt_module.add_files()
                bt_module_list.append(bt_module)
            except Exception as e:
                print(e)
                continue

        # Add relations between the modules (parent and child nodes)
        for module in bt_module_list:
            for parent_module in bt_module_list:
                if module == parent_module:
                    continue
                if parent_module.path == os.path.dirname(module.path):
                    parent_module.child_module.append(module)
                    module.parent_module = parent_module

        # Find the root node
        self.root_module = next(
            filter(lambda e: e.parent_module is None, bt_module_list)
        )
        self.base_module = self.root_module

        # Add dependencies between all the files
        btf_map = self.get_all_bt_files_map()

        for bt_file in btf_map.values():
            imported_modules = get_imported_modules(
                bt_file.ast, self.target_project_base_location, self.am
            )
            bt_file >> [
                btf_map[module.file]
                for module in imported_modules
                if module.file in btf_map
            ]

        sys.path = sys.path[2:]
        astroid.manager.AstroidManager().clear_cache()
        self.am.clear_cache()

    def get_bt_file(self, path: str) -> BTFile:
        file_path = self.am.ast_from_module_name(path).file
        bt_file = self.get_all_bt_files_map()[file_path]
        return bt_file

    def get_bt_module(self, path: str) -> BTModule:
        path_list = path.split(".")[1:]
        current_module = self.root_module
        while path_list:
            current_module = next(
                filter(
                    lambda e: e.name == path_list[0],
                    current_module.child_module,
                )
            )
            path_list.pop(0)
        return current_module

    def change_scope(self, path: str):
        self.root_module = self.get_bt_module(path)

    def get_all_bt_files_map(self) -> dict[str, BTFile]:
        return {
            btf.file: btf for btf in self.root_module.get_files_recursive()
        }

    def get_all_bt_modules_map(self) -> dict[str, BTModule]:
        return {
            btm.path: btm
            for btm in self.root_module.get_submodules_recursive()
        }

    def _get_files_recursive(self, path: str) -> list[str]:
        file_list = []
        t = list(os.walk(path))
        for root, _, files in t:
            for file in files:
                file_list.append(os.path.join(root, file))

        return file_list

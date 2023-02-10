import astroid
import sys
import os

from src.core.bt_file import BTFile, get_imported_modules
from src.core.bt_module import BTModule


class BTGraph:
    DEFAULT_SETTINGS = {"diagram_name": "", "project": None}
    root_module_location: str = None
    target_project_base_location: str = None
    root_module = None
    base_module = None

    def build_graph(self, config_path: str):
        # Read the config file and compile it creating a update and settings function
        source_code = self._get_source_code(config_path)
        self._compile_source_code(source_code, os.path.dirname(config_path))
        self.DEFAULT_SETTINGS.update(settings())

        # Setup the root location of the project
        self.root_module_location = os.path.dirname(
            self.DEFAULT_SETTINGS["project"].__file__
        )
        # Set the base location of the project
        self.target_project_base_location = os.path.dirname(config_path)
        # Add that location to syspath
        sys.path.append(self.root_module_location)

        bt_module_list: list[BTModule] = []

        # Read all the python files within the project
        file_list = self._get_files_recursive(self.root_module_location)

        # Create modules and add them to module list (No relations between them yet)
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

        # Add relations between the modules (parent and child nodes)
        for module in bt_module_list:
            for parent_module in bt_module_list:
                if module == parent_module:
                    continue
                if parent_module.path == "/".join(module.path.split("/")[:-1]):
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
                bt_file.ast, self.target_project_base_location
            )
            bt_file >> [
                btf_map[module.file]
                for module in imported_modules
                if module.file in btf_map
            ]

        # Call the update app from the provided config file, to create constraints
        update(self)

    def get_bt_file(self, path: str) -> BTFile:
        file_path = astroid.MANAGER.ast_from_module_name(path).file
        bt_file = self.get_all_bt_files_map()[file_path]
        return bt_file

    def get_bt_module(self, path: str) -> BTModule:
        path_list = path.split(".")[1:]
        current_module = self.root_module
        while path_list:
            current_module = next(
                filter(lambda e: e.name == path_list[0], current_module.child_module)
            )
            path_list.pop(0)
        return current_module

    def change_scope(self, path: str):
        self.root_module = self.get_bt_module(path)

    def get_all_bt_files_map(self) -> dict[str, BTFile]:
        return {btf.file: btf for btf in self.root_module.get_files_recursive()}

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


def setup():
    pass  # overridden by config file


def settings():
    pass  # overridden by config file


def update():
    pass  # overridden by config file

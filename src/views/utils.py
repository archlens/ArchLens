from src.core.bt_module import BTModule
from src.core.bt_file import BTFile
from src.utils.path_manager_singleton import PathManagerSingleton


def get_view_package_path_from_bt_package(bt_package: BTModule | BTFile) -> str:
    path_manager = PathManagerSingleton()
    if isinstance(bt_package, BTFile):
        # For files, use the file path without .py extension
        raw_name = path_manager.get_relative_path_from_project_root(bt_package.path, True)
        return raw_name.replace(".py", "")
    raw_name = path_manager.get_relative_path_from_project_root(bt_package.path, True)
    return raw_name

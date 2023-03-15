from src.core.bt_module import BTModule
from src.utils.path_manager_singleton import PathManagerSingleton


def get_pu_package_path_from_bt_package(bt_package: BTModule) -> str:
    path_manager = PathManagerSingleton()
    raw_name = path_manager.get_relative_path_from_project_root(
        bt_package.path, True
    )
    return raw_name

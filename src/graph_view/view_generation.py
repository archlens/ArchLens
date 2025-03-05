from astroid.manager import AstroidManager

from src.core.bt_graph import BTGraph
from src.core.bt_module import BTModule


class ViewGenerator:
    astroid_manager: AstroidManager = None

    def __init__(self, astroid_manager: AstroidManager):
        self.astroid_manager = astroid_manager


def generate_views(graph: BTGraph, config: dict) -> dict[str, list[BTModule]]:
    bt_packages = graph.get_all_bt_modules_map()
    result_views = {}

    for view_name, view_config in config["views"].items():
        filtered_packages = []

        for bt_package in bt_packages.values():
            if module_in_view(view_config, bt_package):
                filtered_packages.append(bt_package)

        result_views[view_name] = filtered_packages

    return result_views

def module_in_view(view: dict, module: BTModule) -> bool:
    packages = view["packages"]

    if len(packages) is 0:
        return False
    else:
        return False

    return False
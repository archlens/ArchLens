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


def _filter_packages(
    packages_map: dict[str, PuPackage], view: dict
) -> dict[str, PuPackage]:
    packages = list(packages_map.values())
    filtered_packages_set: set[PuPackage] = set()
    # packages
    for package_view in view["packages"]:
        for package in packages:
            filter_path = package_view

            if isinstance(package_view, str):
                if package.path.startswith(filter_path.replace(".", "/")):
                    filtered_packages_set.add(package)

            if isinstance(package_view, dict):
                filter_path = package_view["path"].replace(".", "/")
                filter_path = filter_path.replace("*", "")
                view_depth = package_view["depth"]
                if filter_path == "" and package.parent_path == ".":
                    filtered_packages_set.add(package)
                    depth_filter_packages = _find_packages_with_depth(
                        package, view_depth - 1, packages_map
                    )
                    filtered_packages_set.update(depth_filter_packages)
                elif package.path == filter_path:
                    filtered_packages_set.add(package)
                    depth_filter_packages = _find_packages_with_depth(
                        package, view_depth, packages_map
                    )
                    filtered_packages_set.update(depth_filter_packages)

    if len(view["packages"]) == 0:
        filtered_packages_set = set(packages_map.values())







    # ignorePackages
    updated_filtered_packages_set: set = set()
    for ignore_packages in view["ignorePackages"]:
        for package in filtered_packages_set:
            should_filter = False
            if ignore_packages.startswith("*") and ignore_packages.endswith("*"):
                if ignore_packages[1:-1] in package.path:
                    should_filter = True
            else:
                if package.path.startswith(ignore_packages):
                    should_filter = True

            if not should_filter:
                updated_filtered_packages_set.add(package)

    if len(view["ignorePackages"]) == 0:
        updated_filtered_packages_set = filtered_packages_set

    filtered_packages_set = updated_filtered_packages_set

    for package in filtered_packages_set:
        package.filter_excess_packages_dependencies(filtered_packages_set)
    return {package.path: package for package in filtered_packages_set}

def _find_packages_with_depth(
    package: PuPackage, depth: int, pu_package_map: dict[str, PuPackage]
):
    bt_sub_packages = package.bt_package.get_submodules_recursive()
    filtered_sub_packages = [
        get_pu_package_path_from_bt_package(sub_package)
        for sub_package in bt_sub_packages
        if (sub_package.depth - package.bt_package.depth) <= depth
    ]
    return [pu_package_map[p] for p in filtered_sub_packages]
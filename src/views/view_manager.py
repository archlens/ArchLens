import sys
from src.core.bt_graph import BTGraph
from src.views.view_entities import (
    PACKAGE_NAME_SPLITTER,
    EntityState,
    ViewPackage,
)
from src.views.utils import get_view_package_path_from_bt_package
import os
from typing import Callable


def render_views(
    graph: BTGraph,
    config: dict,
    save_to_file: Callable[[list[ViewPackage], str, dict], None],
):
    views = _create_view_graphs(graph, config)
    for view_name, view_package_map in views.items():
        if os.getenv("MT_DEBUG"):
            dep_count = sum(
                len(package.view_dependency_list)
                for package in view_package_map.values()
            )
            package_count = len(list(view_package_map.values()))
            print("View name:", view_name)
            print("Package count:", package_count)
            print("Dependency count:", dep_count)

        view_graph = list(view_package_map.values())
        view_config: dict = config["views"][view_name]
        use_package_path_as_label = view_config.get("usePackagePathAsLabel", True)
        if not use_package_path_as_label:
            _handle_duplicate_name(view_graph)

        save_to_file(view_graph, view_name, config)


def render_diff_views(
    local_bt_graph: BTGraph,
    remote_bt_graph: BTGraph,
    config: dict,
    save_to_file: Callable[[list[ViewPackage], str, dict], None],
) -> set[str]:
    """
    Renders diff views between local and remote graphs.

    Returns:
        set[str]: Set of view names that have architectural changes.
    """
    changed_views: set[str] = set()
    local_graph_views = _create_view_graphs(local_bt_graph, config)
    remote_graph_views = _create_view_graphs(remote_bt_graph, config)

    for view_name, local_graph in local_graph_views.items():
        diff_graph: list[ViewPackage] = []
        remote_graph = remote_graph_views[view_name]
        packages_to_skip_dependency_update: set[str] = set()
        view_has_changes = False

        # Created packages
        for path, package in local_graph.items():
            if path not in remote_graph:
                package.state = EntityState.CREATED
                for package_dependency in package.view_dependency_list:
                    package_dependency.state = EntityState.CREATED
                diff_graph.append(package)
                view_has_changes = True

        # Deleted packages
        for remote_path, remote_package in remote_graph.items():
            if remote_path not in local_graph:
                remote_package.state = EntityState.DELETED
                for remote_package_dependencies in remote_package.view_dependency_list:
                    remote_package_dependencies.state = EntityState.DELETED
                diff_graph.append(remote_package)
                local_graph[remote_path] = remote_package
                packages_to_skip_dependency_update.add(remote_path)
                view_has_changes = True

        # Change dependency state
        for path, package in local_graph.items():
            if path not in remote_graph or path in packages_to_skip_dependency_update:
                continue  # We have already dealt with this case above
            local_dependency_map = package.get_dependency_map()
            remote_dependency_map = remote_graph[path].get_dependency_map()
            for remote_key, remote_value in remote_dependency_map.items():
                # Check if the same key exists in the local_dependency_map
                if remote_key not in local_dependency_map:
                    # we have a dependency that no longer exists in local, package exists but the dependency removed
                    color = EntityState.DELETED
                    dependency_count = 0 - remote_value.dependency_count
                    remote_value = remote_dependency_map[remote_key]

                    remote_dependency_map[remote_key].render_diff = {
                        "from_package": remote_value.from_package,
                        "to_package": remote_value.to_package,
                        "color": color,
                        "label": f"0 ({dependency_count})",
                    }
                    view_has_changes = True
                    continue

                local_value = local_dependency_map[remote_key]

                # Check if dependency counts are different
                if remote_value.dependency_count != local_value.dependency_count:
                    diff = local_value.dependency_count - remote_value.dependency_count
                    sign = "+" if diff > 0 else ""
                    color = EntityState.CREATED if diff > 0 else EntityState.DELETED
                    dependency_count = (
                        f"{local_value.dependency_count} ({sign}{diff})"
                        if diff != 0
                        else f"{local_value.dependency_count}"
                    )

                    local_dependency_map[remote_key].render_diff = {
                        "from_package": local_value.from_package,
                        "to_package": local_value.to_package,
                        "color": color,
                        "label": f"{dependency_count}",
                    }
                    view_has_changes = True

            # Created dependencies
            for dependency_path, dependency in local_dependency_map.items():
                if dependency_path not in remote_dependency_map:
                    # we treat a new dependency as a diff
                    color = EntityState.CREATED
                    dependency_count = dependency.dependency_count
                    dependency.render_diff = {
                        "from_package": dependency.from_package,
                        "to_package": dependency.to_package,
                        "color": color,
                        "label": f"{dependency_count} (+{dependency_count})",
                    }
                    view_has_changes = True

            # Deleted dependencies
            for (
                remote_dependency_path,
                remote_dependency,
            ) in remote_dependency_map.items():
                if remote_dependency_path not in local_dependency_map:
                    remote_dependency.state = EntityState.DELETED
                    remote_dependency.from_package = package
                    remote_dependency.to_package = local_graph[
                        remote_dependency_path
                    ]  # Ensures that the package refs will be in the final graph
                    package.view_dependency_list.append(remote_dependency)
                    view_has_changes = True

            diff_graph.append(package)

        view_config: dict = config["views"][view_name]
        use_package_path_as_label = view_config.get("usePackagePathAsLabel", True)
        if not use_package_path_as_label:
            _handle_duplicate_name(diff_graph)

        if view_has_changes:
            changed_views.add(view_name)
            save_to_file(diff_graph, view_name, config)

    return changed_views


def _handle_duplicate_name(view_graph: list[ViewPackage]):
    for package in view_graph:
        package_name_split = package.path.split("/")
        found_duplicate = False
        for package_2 in view_graph:
            if package == package_2:
                continue
            package_2_name_split = package_2.path.split("/")
            if package_2_name_split[-1] == package_name_split[-1]:
                if len(package_name_split) >= len(package_2_name_split):
                    package.name = PACKAGE_NAME_SPLITTER.join(package_name_split[-2:])
                else:
                    package.name = package_name_split[-1]
                found_duplicate = True

        if not found_duplicate:
            package.name = package_name_split[-1]


def _create_view_graphs(
    graph: BTGraph, config: dict
) -> dict[str, dict[str, ViewPackage]]:

    # all the nodes at all kinds of lelves
    bt_packages = graph.get_all_bt_modules_map()
    views = {}

    for view_name, view in config["views"].items():

        viewpackages_by_name: dict[str, ViewPackage] = {}

        for bt_package in bt_packages.values():

            # create a view package node for each of the nodes in the AST
            view_package = ViewPackage(bt_package)
            viewpackages_by_name[view_package.path] = view_package

        for view_package in viewpackages_by_name.values():
            view_package.setup_dependencies(viewpackages_by_name)

        viewpackages_by_name = _filter_packages(viewpackages_by_name, view)
        views[view_name] = viewpackages_by_name
    return views


def _find_packages_with_depth(
    package: ViewPackage, depth: int, view_package_map: dict[str, ViewPackage]
):

    bt_sub_packages = package.bt_package.get_submodules_recursive()
    filtered_sub_packages = [
        get_view_package_path_from_bt_package(sub_package)
        for sub_package in bt_sub_packages
        if (sub_package.depth - package.bt_package.depth) == depth
    ]
    return [view_package_map[p] for p in filtered_sub_packages]


def _filter_packages(
    packages_map: dict[str, ViewPackage], view: dict
) -> dict[str, ViewPackage]:
    all_viewpackages = list(packages_map.values())
    filtered_packages_set: set[ViewPackage] = set()

    # packages
    for package_definition_from_config in view["packages"]:
        for view_package in all_viewpackages:
            filter_path = package_definition_from_config

            # e.g.
            # "packages": [
            #      "api",
            # ],
            if isinstance(package_definition_from_config, str):
                if view_package.path.startswith(filter_path.replace(".", "/")):
                    filtered_packages_set.add(view_package)

            # e.g.
            # "packages": [
            #     {
            #         "path": "api",
            #         "depth": 1
            #     }
            # ],
            if isinstance(package_definition_from_config, dict):
                filter_path = package_definition_from_config["path"].replace(".", "/")

                # TODO: why are we ignoring the star - it's a regex?
                filter_path = filter_path.replace("*", "")
                view_depth = package_definition_from_config["depth"]

                if filter_path == "" and view_package.is_root_package():
                    # This happens when config specifies {"path": "", "depth": N}
                    # to include all root packages up to depth N
                    filtered_packages_set.add(view_package)

                    depth_filter_packages = _find_packages_with_depth(
                        view_package, view_depth - 1, packages_map
                    )
                    filtered_packages_set.update(depth_filter_packages)
                elif view_package.path == filter_path:

                    if view_depth == 0:
                        # if view depth is greater, that means we want to expand this path
                        # and this path should not be part of the view
                        filtered_packages_set.add(view_package)

                    depth_filter_packages = _find_packages_with_depth(
                        view_package, view_depth, packages_map
                    )
                    filtered_packages_set.update(depth_filter_packages)

    if len(view["packages"]) == 0:
        # If no packages specified, only include root packages (those without a parent)
        filtered_packages_set = set(
            package for package in packages_map.values() 
            if package.is_root_package()
        )

    # ignorePackages

    if not "ignorePackages" in view:
        view["ignorePackages"] = []

    updated_filtered_packages_set: set = set()
    for view_package in filtered_packages_set:
        should_filter = False

        for ignore_packages in view["ignorePackages"]:
            ignore_packages = ignore_packages.replace(".", "/")
            if ignore_packages.startswith("*") and ignore_packages.endswith("*"):
                if ignore_packages[1:-1] in view_package.path:
                    should_filter = True
            else:
                if view_package.path.startswith(ignore_packages):
                    should_filter = True

        if not should_filter:
            updated_filtered_packages_set.add(view_package)

    if len(view["ignorePackages"]) == 0:
        updated_filtered_packages_set = filtered_packages_set

    filtered_packages_set = updated_filtered_packages_set

    for view_package in filtered_packages_set:
        view_package.filter_excess_packages_dependencies(filtered_packages_set)
    return {package.path: package for package in filtered_packages_set}

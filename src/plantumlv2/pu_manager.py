import sys
from src.core.bt_graph import BTGraph
from src.plantumlv2.pu_entities import (
    PACKAGE_NAME_SPLITTER,
    EntityState,
    PuPackage,
)
from src.plantumlv2.utils import get_pu_package_path_from_bt_package
import os


def render_pu(graph: BTGraph, config: dict):
    views = _create_pu_graph(graph, config)
    for view_name, pu_package_map in views.items():
        plant_uml_str = _render_pu_graph(
            list(pu_package_map.values()), view_name, config
        )
        project_name = config["name"]
        save_location = os.path.join(
            config["saveLocation"], f"{project_name}-{view_name}"
        )
        _save_plantuml_str(save_location, plant_uml_str)


def render_diff_pu(
    local_bt_graph: BTGraph, remote_bt_graph: BTGraph, config: dict
):
    project_name = config["name"]
    local_graph_views = _create_pu_graph(local_bt_graph, config)
    remote_graph_views = _create_pu_graph(remote_bt_graph, config)
    packages_to_skip_dependency_update = set()

    for view_name, local_graph in local_graph_views.items():
        diff_graph: list[PuPackage] = []
        remote_graph = remote_graph_views[view_name]
        # Created packages
        for path, package in local_graph.items():
            if path not in remote_graph:
                package.state = EntityState.CREATED
                for package_dependency in package.pu_dependency_list:
                    package_dependency.state = EntityState.CREATED
                diff_graph.append(package)

        # Deleted packages
        for remote_path, remote_package in remote_graph.items():
            if remote_path not in local_graph:
                remote_package.state = EntityState.DELETED
                for (
                    remote_package_dependencies
                ) in remote_package.pu_dependency_list:
                    remote_package_dependencies.state = EntityState.DELETED
                diff_graph.append(remote_package)
                local_graph[remote_path] = remote_package
                packages_to_skip_dependency_update.add(remote_path)

        # Change dependency state
        for path, package in local_graph.items():
            if (
                path not in remote_graph
                or path in packages_to_skip_dependency_update
            ):
                continue  # We have already dealt with this case above
            local_dependency_map = package.get_dependency_map()
            remote_dependency_map = remote_graph[path].get_dependency_map()

            # Created dependencies
            for dependency_path, dependency in local_dependency_map.items():
                if dependency_path not in remote_dependency_map:
                    dependency.state = EntityState.CREATED

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
                    package.pu_dependency_list.append(remote_dependency)

            diff_graph.append(package)

        plant_uml_str = _render_pu_graph(diff_graph, view_name, config)
        save_location = os.path.join(
            config["saveLocation"], f"{project_name}-diff-{view_name}"
        )
        _save_plantuml_str(save_location, plant_uml_str)


def _handle_duplicate_name(pu_graph: list[PuPackage]):
    for package in pu_graph:
        package_name_split = package.path.split("/")
        found_duplicate = False
        for package_2 in pu_graph:
            if package == package_2:
                continue
            package_2_name_split = package_2.path.split("/")
            if package_2_name_split[-1] == package_name_split[-1]:
                if len(package_name_split) >= len(package_2_name_split):
                    package.name = PACKAGE_NAME_SPLITTER.join(
                        package_name_split[-2:]
                    )
                else:
                    package.name = package_name_split[-1]
                found_duplicate = True

        if not found_duplicate:
            package.name = package_name_split[-1]


def _render_pu_graph(pu_graph: list[PuPackage], view_name, config):
    view_config: dict = config["views"][view_name]
    use_package_path_as_label = view_config.get("usePackagePathAsLabel", True)
    if not use_package_path_as_label:
        _handle_duplicate_name(pu_graph)
    pu_package_string = "\n".join(
        [pu_package.render_package() for pu_package in pu_graph]
    )
    pu_dependency_string = "\n".join(
        [pu_package.render_dependency() for pu_package in pu_graph]
    )
    project_name = config.get("name", "")
    title = f"{project_name}-{view_name}"
    uml_str = f"""
@startuml
skinparam backgroundColor GhostWhite
title {title}
{pu_package_string}
{pu_dependency_string}
@enduml
        """

    if os.getenv("MT_DEBUG"):
        print(uml_str)
        print("Program Complete")
    return uml_str


def _save_plantuml_str(file_name: str, data: str):
    os.makedirs(os.path.dirname(file_name), exist_ok=True)
    with open(file_name, "w") as f:
        f.write(data)
    python_executable = sys.executable
    plantuml_server = os.getenv(
        "PLANTUML_SERVER_URL",
        "https://mt-plantuml-app-service.azurewebsites.net/img/",
    )
    os.system(
        f"{python_executable} -m plantuml --server {plantuml_server}  {file_name}"
    )

    if os.path.exists(file_name):
        os.remove(file_name)


def _create_pu_graph(
    graph: BTGraph, config: dict
) -> dict[str, dict[str, PuPackage]]:
    bt_packages = graph.get_all_bt_modules_map()
    views = {}

    for view_name, view in config["views"].items():
        pu_package_map: dict[str, PuPackage] = {}
        for bt_package in bt_packages.values():
            pu_package = PuPackage(bt_package)
            pu_package_map[pu_package.path] = pu_package

        for pu_package in pu_package_map.values():
            pu_package.setup_dependencies(pu_package_map)

        pu_package_map = _filter_packages(pu_package_map, view)
        views[view_name] = pu_package_map
    return views


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
                filter_path = package_view["packagePath"].replace(".", "/")
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
            if ignore_packages.startswith("*") and ignore_packages.endswith(
                "*"
            ):
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

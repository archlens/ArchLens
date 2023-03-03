import os
from src.core.bt_graph import BTGraph


def verify_config_options(config: dict, graph: BTGraph):
    views = config.get("views", {})
    root_path = os.path.join(config["_config_path"], config["rootFolder"])
    modules = graph.get_all_bt_modules_map()
    for view_data in views.values():
        packages = view_data.get("packages")
        ignore_packages = [
            element
            for element in view_data.get("ignorePackages")
            if "*" not in element
        ]

        total_packages = set((packages + ignore_packages))
        for package in total_packages:
            t = os.path.join(root_path, package)
            if t not in modules:
                raise Exception(f"{package} could not be found")

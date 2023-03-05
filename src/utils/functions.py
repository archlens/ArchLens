import os
from src.core.bt_graph import BTGraph


def verify_config_options(config: dict, graph: BTGraph):
    views = config.get("views", {})
    root_path = os.path.join(config["_config_path"], config["rootFolder"])
    modules = graph.get_all_bt_modules_map()
    for view_data in views.values():
        packages = view_data.get("packages")
        ignore_packages = [
            element for element in view_data.get("ignorePackages") if "*" not in element
        ]

        total_packages = packages + ignore_packages
        if os.name == "nt":
            total_packages = [element.replace("/", "\\") for element in total_packages]

        for package in total_packages:

            if type(package) == str:
                t = os.path.join(root_path, package)
                if package not in root_path:
                    if t not in modules:
                        raise Exception(
                            f"{package} package from config file does not exist in project"
                        )
            else:
                t = os.path.join(root_path, package["packagePath"])
                if package["packagePath"] not in root_path:
                    name = package["packagePath"]
                    if t not in modules:
                        raise Exception(
                            f"{name} package from config file does not exist in project"
                        )

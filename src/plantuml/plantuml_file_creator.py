import os
from src.core.bt_module import BTModule
from pathlib import Path
import fileinput
import sys
from src.utils.path_manager_singleton import PathManagerSingleton
from src.utils.config_manager_singleton import ConfigManagerSingleton

# list of subdomains is a set of strings, could be:
# "test_project/tp_src/api"
# "test_project/tp_src/tp_core/tp_sub_core"
# this would give the 2 sub-systems starting at api and tp_sub_core and how/if they relate in a drawing


def plantuml_diagram_creator_sub_domains(
    root_node,
    diagram_name,
    packages,
    ignore_packages,
    compare_graph_root,
    root_folder,
    path_view,
    save_location="./",
):
    create_directory_if_not_exist(save_location)

    global root_module
    root_module = root_folder

    diagram_type = "package "
    diagram_name = diagram_name.replace(" ", "_")
    diagram_name_txt = ""
    if compare_graph_root is not None:
        diagram_name_txt = save_location + diagram_name + "diffView"
    else:
        diagram_name_txt = save_location + diagram_name

    diagram_name_txt = diagram_name_txt + ".txt"

    que: Queue[BTModule] = Queue()
    que.enqueue(root_node)

    # tracks paths of nodes, so we dont enter the same node twice, path is needed so we dont hit duplicates
    node_tracker = {}

    # keeps track of names so we dont duplicate name modules in the graph
    name_tracker = {}

    if os.path.exists(diagram_name_txt):
        os.remove(diagram_name_txt)

    # adding root to the drawing IF its meant to be in there

    if check_if_module_should_be_in_filtered_graph(root_node, packages):
        f = open(diagram_name_txt, "a")
        f.write("@startuml \n")
        f.write("skinparam backgroundColor GhostWhite" + "\n")
        f.write(
            "title **Title**: "
            + diagram_name
            + ". **Rootfolder**: "
            + root_folder
            + "\n"
        )
        f.close()
    else:
        f = open(diagram_name_txt, "a")
        f.write("@startuml \n")
        f.write("skinparam backgroundColor GhostWhite" + "\n")
        f.write("title " + diagram_name + " \n")
        f.close()

    while not que.isEmpty():
        curr_node: BTModule = que.dequeue()

        # adds all modules we want in our subgraph
        for child in curr_node.child_module:
            if child.path not in node_tracker:
                duplicate_name_check(name_tracker, child, node_tracker, None, True)
                que.enqueue(child)
                node_tracker[child.path] = child
                name_tracker[child.name] = child

                if not ignore_modules_check(ignore_packages, child.path):
                    if check_if_module_should_be_in_filtered_graph(child, packages):
                        config_manager = ConfigManagerSingleton()
                        package_color = config_manager.package_color
                        f = open(diagram_name_txt, "a")
                        f.write(
                            diagram_type
                            + '"'
                            + get_name_for_module_duplicate_checker(child, path_view)
                            + f'"#{package_color}'
                            + "\n"
                        )
                        f.close()

    # adding all dependencies
    que.enqueue(root_node)
    node_tracker_dependencies = {}
    dependencies_map = {}

    while not que.isEmpty():
        curr_node: BTModule = que.dequeue()

        for child in curr_node.child_module:
            if child.path not in node_tracker_dependencies:
                que.enqueue(child)
                node_tracker_dependencies[child.path] = True

        dependencies: set[BTModule] = curr_node.get_module_dependencies()
        name_curr_node = get_name_for_module_duplicate_checker(curr_node, path_view)

        if not ignore_modules_check(ignore_packages, curr_node.path):
            for dependency in dependencies:
                if not ignore_modules_check(ignore_packages, dependency.path):
                    name_dependency = get_name_for_module_duplicate_checker(
                        dependency, path_view
                    )
                    if check_if_module_should_be_in_filtered_graph(
                        dependency, packages
                    ) and check_if_module_should_be_in_filtered_graph(
                        curr_node, packages
                    ):
                        # this if statement is made so that we dont point to ourselves
                        if name_curr_node != name_dependency:
                            # used to detect dependency changes
                            if name_curr_node in dependencies_map:
                                dependency_list: list = dependencies_map[name_curr_node]
                                dependency_list.append(dependency)
                                dependencies_map[name_curr_node] = dependency_list
                            else:
                                dependency_list = []
                                dependency_list.append(dependency)
                                dependencies_map[name_curr_node] = dependency_list

                            f = open(diagram_name_txt, "a")
                            f.write(
                                '"'
                                + name_curr_node
                                + '"'
                                + "-->"
                                + '"'
                                + name_dependency
                                + f'"{get_dependency_string(curr_node, dependency)}'
                                + "\n"
                            )
                            f.close()

    # here we decide if we are diff view
    if compare_graph_root is not None:
        diff_checker = False

        ##################################### #########################################
        # this section finds modules and colors them green or red depending on if theyre old or new

        que.enqueue(compare_graph_root)
        bfs_node_tracker = {}
        main_nodes = {}
        while not que.isEmpty():
            curr_node: BTModule = que.dequeue()
            for child in curr_node.child_module:
                if child.path not in bfs_node_tracker:
                    que.enqueue(child)
                    bfs_node_tracker[child.path] = True
                    if not ignore_modules_check(ignore_packages, child.path):
                        duplicate_name_check(
                            main_nodes, child, node_tracker, root_folder, True
                        )
                        if check_if_module_should_be_in_filtered_graph(child, packages):
                            main_nodes[child.name] = child

                            # this will be true, if the package has been deleted
                            if (
                                child.name not in name_tracker
                                and not ignore_modules_check(
                                    ignore_packages, child.path
                                )
                            ):
                                diff_checker = True
                                f = open(diagram_name_txt, "a")
                                f.write(
                                    diagram_type
                                    + '"'
                                    + get_name_for_module_duplicate_checker(
                                        child, path_view
                                    )
                                    + '" #red'
                                    + "\n"
                                )
                                f.close()

        for child in name_tracker.values():
            # children from original graph
            if check_if_module_should_be_in_filtered_graph(
                child, packages
            ) and not ignore_modules_check(ignore_packages, child.path):
                node: BTModule = child
                name = node.name
                if name not in main_nodes:
                    diff_checker = True
                    f = open(diagram_name_txt, "a")
                    f.write(
                        diagram_type
                        + '"'
                        + get_name_for_module_duplicate_checker(node, path_view)
                        + '" #green'
                        + "\n"
                    )
                    f.close()

        ##############################################################################
        # this section finds Dependencies and colors them green or red depending on if theyre old or new

        que.enqueue(compare_graph_root)

        dependencies_map_main_graph = {}

        node_tracker_dependencies = {}
        while not que.isEmpty():
            curr_node: BTModule = que.dequeue()
            for child in curr_node.child_module:
                if child.path not in node_tracker_dependencies:
                    que.enqueue(child)
                    node_tracker_dependencies[child.path] = True

            if not ignore_modules_check(ignore_packages, curr_node.path):
                name_curr_node = get_name_for_module_duplicate_checker(
                    curr_node, path_view
                )
                dependencies: set[BTModule] = curr_node.get_module_dependencies()

                dependencies_map_main_graph[name_curr_node] = dependencies

                list_of_red_dependencies = find_red_dependencies(
                    dependencies_map, name_curr_node, dependencies
                )

                for dependency in list_of_red_dependencies:
                    if not ignore_modules_check(ignore_packages, dependency.path):
                        name_dependency = get_name_for_module_duplicate_checker(
                            dependency, path_view
                        )
                        if check_if_module_should_be_in_filtered_graph(
                            dependency, packages
                        ) and check_if_module_should_be_in_filtered_graph(
                            curr_node, packages
                        ):
                            # this if statement is made so that we dont point to ourselves
                            if name_curr_node != name_dependency:
                                diff_checker = True
                                f = open(diagram_name_txt, "a")
                                f.write(
                                    '"'
                                    + name_curr_node
                                    + '"'
                                    + "-->"
                                    + '"'
                                    + name_dependency
                                    + f'" #red {get_dependency_string(curr_node, dependency)}'
                                    + "\n"
                                )
                                f.close()

        for dependency in dependencies_map:
            # api, and a list of all its dependencies (from the base graph)
            list_of_new_dependencies = dependencies_map[dependency]

            list_of_old_dependencies = []
            if dependency in dependencies_map_main_graph:
                list_of_old_dependencies = dependencies_map_main_graph[dependency]

            for dep in list_of_new_dependencies:
                dep_name = get_name_for_module_duplicate_checker(dep, path_view)
                if not path_view:
                    dep_name = dep.name

                not_found_partner = True
                for dep_old in list_of_old_dependencies:
                    if dep.name == dep_old.name:
                        not_found_partner = False
                        break

                if not_found_partner:
                    diff_checker = True
                    for line in fileinput.input(diagram_name_txt, inplace=True):
                        print(
                            line.replace(  # TOOD: this replace does not work
                                f'"{dependency}"-->"{dep_name}"',
                                f'"{dependency}"-->"{dep_name}" #green',
                            ),
                            end="",
                        )

    #########################################################################################################
    # ends the uml

    f = open(diagram_name_txt, "a")
    f.write("@enduml")
    f.close()

    if compare_graph_root is not None and diff_checker:
        create_file(diagram_name_txt)
    elif compare_graph_root is None:
        create_file(diagram_name_txt)

    if not os.getenv("MT_DEBUG"):
        os.remove(diagram_name_txt)


def find_red_dependencies(new_dependencies, node_name, old_dependencies: set[BTModule]):
    res = []

    if node_name not in new_dependencies:
        for dependency in old_dependencies:
            res.append(dependency)
    else:
        list_of_new_dep_graph = new_dependencies[node_name]
        for dependency in old_dependencies:
            not_found_partner = True
            for new_dependency in list_of_new_dep_graph:
                if dependency.name == new_dependency.name:
                    not_found_partner = False
                    break
            if not_found_partner:
                res.append(dependency)
    return res


def create_file(name):
    python_executable = sys.executable
    plantuml_server = os.getenv(
        "PLANTUML_SERVER_URL",
        "https://mt-plantuml-app-service.azurewebsites.net/img/",
    )
    os.system(f"{python_executable} -m plantuml --server {plantuml_server}  {name}")


def get_name_for_module_duplicate_checker(module: BTModule, path):
    if path:
        path_manager = PathManagerSingleton()
        module_split = path_manager.get_relative_path_from_project_root(
            module.path, True
        )
        module_split = module_split.replace("/", ".")
        return module_split
    if module.name_if_duplicate_exists is not None:
        module.name_if_duplicate_exists = module.name_if_duplicate_exists.replace(
            "/", "."
        )
        return module.name_if_duplicate_exists
    return module.name


def duplicate_name_check(
    node_names,
    curr_node: BTModule,
    path_tracker,
    root_folder=None,
    first=False,
):
    if was_node_in_original_graph(curr_node, path_tracker, root_folder):
        if not curr_node.name_if_duplicate_exists:
            if curr_node.name in node_names:
                curr_node_split = split_path(curr_node.path)
                curr_node_name = curr_node_split[-2] + "/" + curr_node_split[-1]
                curr_node.name_if_duplicate_exists = curr_node_name
    else:
        if first:
            if curr_node.name in node_names:
                curr_node_split = split_path(curr_node.path)
                curr_node_name = curr_node_split[-2] + "/" + curr_node_split[-1]
                curr_node.name_if_duplicate_exists = curr_node_name


def was_node_in_original_graph(node: BTModule, path_tracker, root_folder):
    if root_folder:
        for path in path_tracker.keys():
            path_split = path.split(root_folder)[-1]
            path_from_tracker = root_folder + path_split
            node_split = node.path.split(root_folder)[-1]
            path_from_node = root_folder + node_split
            if path_from_tracker == path_from_node:
                if len(split_path(path_from_tracker)) != 2:
                    return True
        return False


def ignore_modules_check(list_ignore: list[str], module):
    path_manager = PathManagerSingleton()
    module = path_manager.get_relative_path_from_project_root(module, True)
    for ignore_package in list_ignore:
        if (
            ignore_package.startswith("*")
            and ignore_package.endswith("*")
            and ignore_package[1:-1] in module
        ):
            return True

        if module.startswith(ignore_package):
            return True

    return False


def old_ignore_modules_check(list_ignore, module, root_folder):
    index = module.rindex(root_folder)
    module = module[index:]
    for word in list_ignore:
        if word != "":
            firstChar = word[0]
            lastChar = word[-1]
            # remove any module with word in its path
            if firstChar == "*" and lastChar == "*":
                if word[1:-1] in module:
                    return True
            else:
                # you cant have the module named "core" for instance
                split_mod = split_path(module)
                if split_mod[-1] == word:
                    return True
                # case for if you match directly on a path
                # e.g if you type zeeguu/core, this would remove the zeeguu/core module
                split_mod = split_path(module)
                module = "/".join(split_mod[3:])
                if word == module:
                    return True
    return False


# tp_src/api
def check_if_allowed_module_is_root(allowed_module):
    if type(allowed_module) == str:
        if "/" in allowed_module:
            splits = allowed_module.split("/")
            if splits[0] == root_module and splits[1] == root_module:
                return splits[0]
    else:
        if "/" in allowed_module["packagePath"]:
            splits = allowed_module["packagePath"].split("/")
            if splits[0] == root_module and splits[1] == root_module:
                allowed_module["packagePath"] = splits[0]
                return allowed_module
    return allowed_module


def check_if_module_should_be_in_filtered_graph(module: BTModule, allowed_modules):
    if len(allowed_modules) == 0:
        return True
    path_manager = PathManagerSingleton()
    module_path = path_manager.get_relative_path_from_project_root(module.path)
    for module_curr in allowed_modules:
        module_curr = check_if_allowed_module_is_root(module_curr)
        # this means that we allow all of the sub system, no depth
        if type(module_curr) == str:
            if module_curr in module_path:
                return True
        # if we get here, it means that it is a specified object, at which point we must check for depth
        else:
            path = module_curr["packagePath"]
            depth = module_curr["depth"]
            if path == module_path:
                module.depth = depth
                return True
            else:
                if module.parent_module is not None:
                    if module.parent_module.depth is not None:
                        if path in module_path:
                            if module.parent_module.depth > 0:
                                module.depth = module.parent_module.depth - 1
                                return True

    return False


def create_directory_if_not_exist(path: str):
    Path(path).mkdir(parents=True, exist_ok=True)


def split_path(path) -> list[str]:
    directories = []
    while True:
        path, directory = os.path.split(path)
        if directory:
            directories.append(directory)
        else:
            if path:
                directories.append(path)
            break

    directories.reverse()
    return directories


def get_dependency_string(module: BTModule, dependency_module: BTModule):
    config_manager = ConfigManagerSingleton()
    if config_manager.show_dependency_count:
        dependency_count = module.get_dependency_count(dependency_module)
        return f": {dependency_count}"
    return ""


class Queue:
    def __init__(self):
        self.items = []

    def isEmpty(self):
        return self.items == []

    def enqueue(self, item):
        self.items.insert(0, item)

    def dequeue(self):
        return self.items.pop()

    def size(self):
        return len(self.items)


queue = Queue()

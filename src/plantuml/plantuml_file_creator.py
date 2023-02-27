import os
from src.core.bt_module import BTModule
from pathlib import Path
import fileinput
import sys


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
    save_location="./",
):
    create_directory_if_not_exist(save_location)

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

    if check_if_module_should_be_in_filtered_graph(root_node.path, packages):
        f = open(diagram_name_txt, "a")
        f.write("@startuml \n")
        f.write("title " + diagram_name + "\n")
        f.write(diagram_type + root_node.name + "\n")
        f.close()
    else:
        f = open(diagram_name_txt, "a")
        f.write("@startuml \n")
        f.write("title " + diagram_name + "\n")
        f.close()

    while not que.isEmpty():
        curr_node: BTModule = que.dequeue()

        # adds all modules we want in our subgraph
        for child in curr_node.child_module:
            if child.path not in node_tracker and not ignore_modules_check(
                ignore_packages, child.name
            ):
                duplicate_name_check(
                    name_tracker, child, node_tracker, None, True
                )
                if check_if_module_should_be_in_filtered_graph(
                    child.path, packages
                ):
                    f = open(diagram_name_txt, "a")
                    f.write(
                        diagram_type
                        + '"'
                        + get_name_for_module_duplicate_checker(child)
                        + '"'
                        + "\n"
                    )
                    f.close()

                que.enqueue(child)
                node_tracker[child.path] = child
                # reference the child, so we can add it to the graph in green later if its a new module
                name_tracker[child.name] = child

    # adding all dependencies
    que.enqueue(root_node)
    node_tracker_dependencies = {}
    dependencies_map = {}
    while not que.isEmpty():
        curr_node: BTModule = que.dequeue()

        for child in curr_node.child_module:
            if (
                child.path not in node_tracker_dependencies
                and not ignore_modules_check(ignore_packages, child.name)
            ):
                que.enqueue(child)
                node_tracker_dependencies[child.path] = True

        dependencies: set[BTModule] = curr_node.get_module_dependencies()
        name_curr_node = get_name_for_module_duplicate_checker(curr_node)

        for dependency in dependencies:
            if not ignore_modules_check(ignore_packages, dependency.name):
                name_dependency = get_name_for_module_duplicate_checker(
                    dependency
                )
                if check_if_module_should_be_in_filtered_graph(
                    dependency.path, packages
                ) and check_if_module_should_be_in_filtered_graph(
                    curr_node.path, packages
                ):
                    # this if statement is made so that we dont point to ourselves
                    if name_curr_node != name_dependency:
                        # used to detect dependency changes
                        if name_curr_node in dependencies_map:
                            dependency_list: list = dependencies_map[
                                name_curr_node
                            ]
                            dependency_list.append(dependency)
                            dependencies_map[name_curr_node] = dependency_list
                        else:
                            dependency_list = []
                            dependency_list.append(dependency)
                            dependencies_map[name_curr_node] = dependency_list

                        ##
                        f = open(diagram_name_txt, "a")
                        f.write(
                            '"'
                            + name_curr_node
                            + '"'
                            + "-->"
                            + '"'
                            + name_dependency
                            + '"'
                            + "\n"
                        )
                        f.close()

    # here we decide if we are diff view

    if compare_graph_root is not None:

        ##################################### TO BE REFACTORED #########################################
        # this section finds modules and colors them green or red depending on if theyre old or new

        que.enqueue(compare_graph_root)
        bfs_node_tracker = {}
        main_nodes = {}
        while not que.isEmpty():
            curr_node: BTModule = que.dequeue()
            for child in curr_node.child_module:
                if (
                    child.path not in bfs_node_tracker
                    and not ignore_modules_check(ignore_packages, child.name)
                ):
                    que.enqueue(child)
                    bfs_node_tracker[child.path] = True
                    duplicate_name_check(
                        main_nodes, child, node_tracker, root_folder, True
                    )
                    if check_if_module_should_be_in_filtered_graph(
                        child.path, packages, compare_graph_root
                    ):

                        main_nodes[child.name] = child

                        # this will be true, if the package has been deleted
                        if (
                            child.name not in name_tracker
                            and not ignore_modules_check(
                                ignore_packages, child.name
                            )
                        ):
                            f = open(diagram_name_txt, "a")
                            f.write(
                                diagram_type
                                + '"'
                                + get_name_for_module_duplicate_checker(child)
                                + '" #red'
                                + "\n"
                            )
                            f.close()

        for child in name_tracker.values():
            # children from original graph
            if check_if_module_should_be_in_filtered_graph(
                child.path, packages
            ) and not ignore_modules_check(ignore_packages, child.name):
                node: BTModule = child
                name = node.name
                if name not in main_nodes:
                    f = open(diagram_name_txt, "a")
                    f.write(
                        diagram_type
                        + '"'
                        + get_name_for_module_duplicate_checker(node)
                        + '" #green'
                        + "\n"
                    )
                    f.close()
        ########################################################################################################

        ##################################### TO BE REFACTORED #########################################
        # this section finds Dependencies and colors them green or red depending on if theyre old or new

        que.enqueue(compare_graph_root)

        dependencies_map_main_graph = {}

        node_tracker_dependencies = {}
        while not que.isEmpty():
            curr_node: BTModule = que.dequeue()
            for child in curr_node.child_module:
                name_of_child = get_name_for_module_duplicate_checker(child)
                if (
                    child.path not in node_tracker_dependencies
                    and not ignore_modules_check(
                        ignore_packages, name_of_child
                    )
                ):
                    que.enqueue(child)
                    node_tracker_dependencies[child.path] = True

            name_curr_node = get_name_for_module_duplicate_checker(curr_node)
            dependencies: set[BTModule] = curr_node.get_module_dependencies()

            dependencies_map_main_graph[name_curr_node] = dependencies

            list_of_red_dependencies = find_red_dependencies(
                dependencies_map, name_curr_node, dependencies
            )

            for dependency in list_of_red_dependencies:
                if not ignore_modules_check(ignore_packages, dependency.name):
                    name_dependency = get_name_for_module_duplicate_checker(
                        dependency
                    )
                    if check_if_module_should_be_in_filtered_graph(
                        dependency.path, packages
                    ) and check_if_module_should_be_in_filtered_graph(
                        curr_node.path, packages
                    ):
                        # this if statement is made so that we dont point to ourselves
                        if name_curr_node != name_dependency:
                            f = open(diagram_name_txt, "a")
                            f.write(
                                '"'
                                + name_curr_node
                                + '"'
                                + "-->"
                                + '"'
                                + name_dependency
                                + '" #red'
                                + "\n"
                            )
                            f.close()

        for dependency in dependencies_map:
            # api, and a list of all its dependencies (from the base graph)
            list_of_new_dependencies = dependencies_map[dependency]

            list_of_old_dependencies = []
            if dependency in dependencies_map_main_graph:
                list_of_old_dependencies = dependencies_map_main_graph[
                    dependency
                ]

            for dep in list_of_new_dependencies:
                not_found_partner = True
                for dep_old in list_of_old_dependencies:
                    if dep.name == dep_old.name:
                        not_found_partner = False
                        break
                if not_found_partner:
                    for line in fileinput.input(
                        diagram_name_txt, inplace=True
                    ):
                        print(
                            line.replace(
                                '"'
                                + dependency
                                + '"'
                                + "-->"
                                + '"'
                                + dep.name
                                + '"',
                                '"'
                                + dependency
                                + '"'
                                + "-->"
                                + '"'
                                + dep.name
                                + '" #green',
                            ),
                            end="",
                        )
                    # f = open(diagram_name_txt, "a")
                    # f.write(
                    #     '"' + dependency + '"' + "-->" + '"' + dep.name + '" #green' + "\n"
                    # )
                    # f.close()

    #########################################################################################################
    # ends the uml
    f = open(diagram_name_txt, "a")
    f.write("@enduml")
    f.close()

    create_file(diagram_name_txt)
    # comment in when done, but leaving it in atm for developing purposes
    os.remove(diagram_name_txt)


def find_red_dependencies(
    new_dependencies, node_name, old_dependencies: set[BTModule]
):
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
    os.system(python_executable + " -m plantuml " + name)


def get_name_for_module_duplicate_checker(module: BTModule):
    if module.name_if_duplicate_exists is not None:
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
                curr_node_name = (
                    curr_node_split[-2] + "/" + curr_node_split[-1]
                )
                curr_node.name_if_duplicate_exists = curr_node_name
    else:
        if first:
            if curr_node.name in node_names:
                curr_node_split = split_path(curr_node.path)
                curr_node_name = (
                    curr_node_split[-2] + "/" + curr_node_split[-1]
                )
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


def ignore_modules_check(list_ignore, module):
    for word in list_ignore:
        if word in module:
            return True
    return False


def check_if_module_should_be_in_filtered_graph(
    module, allowed_modules, compare_graph_root=None
):
    if len(allowed_modules) == 0:
        return True
    if compare_graph_root is not None:
        split_mod = split_path(module)
        module = "/".join(split_mod[3:])
    for module_curr in allowed_modules:
        if module_curr in module:
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

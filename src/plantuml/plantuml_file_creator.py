import os
from src.core.bt_module import BTModule
from pathlib import Path


def plantuml_diagram_creator_entire_domain(root_node, diagram_name, save_location="./"):
    create_directory_if_not_exist(save_location)

    print("start")

    diagram_type = "package "

    que: Queue[BTModule] = Queue()

    que.enqueue(root_node)

    node_tracker = {}

    diagram_name = diagram_name.replace(" ", "_")

    diagram_name_txt = save_location + diagram_name + ".txt"

    if os.path.exists(diagram_name_txt):
        os.remove(diagram_name_txt)

    # adding root to the drawing
    f = open(diagram_name_txt, "a")
    f.write("@startuml \n")
    f.write(diagram_type + root_node.name + "\n")
    f.close()

    # adding all modules to the graph
    while que.isEmpty() != True:

        curr_node: BTModule = que.dequeue()

        for child in curr_node.child_module:
            if child.name not in node_tracker:
                f = open(diagram_name_txt, "a")
                f.write(diagram_type + child.name + "\n")
                que.enqueue(child)
                node_tracker[child.name] = True
            f.close()

    # adding all dependencies
    que.enqueue(root_node)
    node_tracker = {}
    while que.isEmpty() != True:

        curr_node: BTModule = que.dequeue()

        for child in curr_node.child_module:
            if child.name not in node_tracker:
                que.enqueue(child)
                node_tracker[child.name] = True

        dependencies: set[BTModule] = curr_node.get_module_dependencies()
        for dependency in dependencies:
            f = open(diagram_name_txt, "a")
            f.write(curr_node.name + "-->" + dependency.name + "\n")
            f.close()

    f = open(diagram_name_txt, "a")
    f.write("@enduml")
    f.close()

    create_file(diagram_name_txt)

    # comment in when done, but leaving it in atm for developing purposes
    # os.remove(diagram_name_txt)


def create_file(name):
    os.system("python -m plantuml " + name)


# list of subdomains is a set of strings, could be:
# "test_project/tp_src/api"
# "test_project/tp_src/tp_core/tp_sub_core"
# this would give the 2 sub-systems starting at api and tp_sub_core and how/if they relate in a drawing
def plantuml_diagram_creator_sub_domains(
    root_node, diagram_name, list_of_subdomains, save_location="./"
):
    create_directory_if_not_exist(save_location)

    diagram_type = "package "
    diagram_name = diagram_name.replace(" ", "_")
    diagram_name_txt = save_location + diagram_name + "_filtered.txt"

    que: Queue[BTModule] = Queue()
    que.enqueue(root_node)

    node_tracker = {}

    if os.path.exists(diagram_name_txt):
        os.remove(diagram_name_txt)

    # adding root to the drawing IF its meant to be in there

    if check_if_module_should_be_in_filtered_graph(root_node.path, list_of_subdomains):
        f = open(diagram_name_txt, "a")
        f.write("@startuml \n")
        f.write(diagram_type + root_node.name + "\n")
        f.close()
    else:
        f = open(diagram_name_txt, "a")
        f.write("@startuml \n")
        f.close()

    while que.isEmpty() != True:

        curr_node = que.dequeue()

        # adds all modules we want in our subgraph
        for child in curr_node.child_module:
            if child.name not in node_tracker:
                if check_if_module_should_be_in_filtered_graph(
                    child.path, list_of_subdomains
                ):
                    f = open(diagram_name_txt, "a")
                    f.write(diagram_type + child.name + "\n")
                    f.close()

                que.enqueue(child)
                node_tracker[child.name] = True

    # adding all dependencies
    que.enqueue(root_node)
    node_tracker = {}
    while que.isEmpty() != True:

        curr_node: BTModule = que.dequeue()

        for child in curr_node.child_module:
            if child.name not in node_tracker:
                que.enqueue(child)
                node_tracker[child.name] = True

        dependencies: set[BTModule] = curr_node.get_module_dependencies()
        for dependency in dependencies:
            x = 4
            if check_if_module_should_be_in_filtered_graph(
                dependency.path, list_of_subdomains
            ) and check_if_module_should_be_in_filtered_graph(
                curr_node.path, list_of_subdomains
            ):
                f = open(diagram_name_txt, "a")
                f.write(curr_node.name + "-->" + dependency.name + "\n")
                f.close()

    # ends the uml
    f = open(diagram_name_txt, "a")
    f.write("@enduml")
    f.close()

    create_file(diagram_name_txt)

    # comment in when done, but leaving it in atm for developing purposes
    # os.remove(diagram_name_txt)


def check_if_module_should_be_in_filtered_graph(module, allowed_modules):
    for module_curr in allowed_modules:
        if module_curr in module:
            return True
    return False


def create_directory_if_not_exist(path: str):
    Path(path).mkdir(parents=True, exist_ok=True)


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

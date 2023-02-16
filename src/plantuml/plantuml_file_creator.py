import os
from src.core.bt_module import BTModule
from pathlib import Path


# list of subdomains is a set of strings, could be:
# "test_project/tp_src/api"
# "test_project/tp_src/tp_core/tp_sub_core"
# this would give the 2 sub-systems starting at api and tp_sub_core and how/if they relate in a drawing
def plantuml_diagram_creator_sub_domains(
    root_node, diagram_name, list_of_subdomains, ignore_modules ,save_location="./"
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

    while not que.isEmpty():

        curr_node: BTModule = que.dequeue()

        # adds all modules we want in our subgraph
        for child in curr_node.child_module:
            if child.path not in node_tracker and not ignore_modules_check(ignore_modules, child.name):
                duplicate_name_check(node_tracker.keys(), child)
                if check_if_module_should_be_in_filtered_graph(
                    child.path, list_of_subdomains
                ):
                    f = open(diagram_name_txt, "a")
                    f.write(diagram_type + "\""+ get_name_for_module_duplicate_checker(child) + "\""+"\n")
                    f.close()

                que.enqueue(child)
                node_tracker[child.path] = True

    # adding all dependencies
    que.enqueue(root_node)
    node_tracker_dependencies = {}
    while not que.isEmpty():

        curr_node: BTModule = que.dequeue()

        for child in curr_node.child_module:
            if child.path not in node_tracker_dependencies and not ignore_modules_check(ignore_modules, child.name):
                que.enqueue(child)
                node_tracker_dependencies[child.path] = True

        dependencies: set[BTModule] = curr_node.get_module_dependencies()
        name_curr_node = get_name_for_module_duplicate_checker(curr_node)

        for dependency in dependencies:
            if not ignore_modules_check(ignore_modules, dependency.name):
                name_dependency = get_name_for_module_duplicate_checker(dependency)
                if check_if_module_should_be_in_filtered_graph(
                    dependency.path, list_of_subdomains
                ) and check_if_module_should_be_in_filtered_graph(
                    curr_node.path, list_of_subdomains
                ):
                    f = open(diagram_name_txt, "a")
                    f.write( "\""+name_curr_node+ "\"" + "-->"+ "\""+  name_dependency +"\"" + "\n")
                    f.close()

    # ends the uml
    f = open(diagram_name_txt, "a")
    f.write("@enduml")
    f.close()

    create_file(diagram_name_txt)

    # comment in when done, but leaving it in atm for developing purposes
    # os.remove(diagram_name_txt)


def create_file(name):
    os.system("python -m plantuml " + name)
    
    
def get_name_for_module_duplicate_checker(module:BTModule):
    if module.name_if_duplicate_exists != None:
        return module.name_if_duplicate_exists
    return module.name

def duplicate_name_check(node_paths, new_node:BTModule):
    for path in node_paths:
        path_sep = path.split("/")
        end_of_path = path_sep[-1]
        if new_node.name == end_of_path:
            new_node_split = new_node.path.split("/")
            new_node_name = "parent:"+new_node_split[-2]+ " * module:"+new_node_split[-1]
            new_node.name_if_duplicate_exists = new_node_name

def ignore_modules_check(list_ignore, module):
    for word in list_ignore:
        if word in module:
            return True
    return False

def check_if_module_should_be_in_filtered_graph(module, allowed_modules):
    if len(allowed_modules) == 0: return True
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

import diagrams

from typing import TYPE_CHECKING


if TYPE_CHECKING:
    from src.core.bt_module import BTModule
    from src.core.bt_graph import BTGraph


def render(graph: "BTGraph"):
    root_module = graph.root_module
    modules = root_module.get_submodules_recursive()
    dependencies_map: dict["BTModule", list["BTModule"]] = {}
    for module in modules:
        dependencies_map[module] = module.get_module_dependencies()

    node_map = {}

    def create_nodes(module: "BTModule"):
        with diagrams.Cluster(label=module.name):
            n = diagrams.Node(label=module.name)
            node_map[module.path] = n
            for child_module in module.child_module:
                create_nodes(child_module)

    create_nodes(root_module)

    for module, dependencies in dependencies_map.items():
        diagram_node = node_map[module.path]
        diagram_node >> [node_map[d.path] for d in dependencies]

    print("test")

from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from src.core.bt_graph import BTGraph


def render(graph: "BTGraph"):
    root_module = graph.root_module
    root_module.get_submodules_recursive
    print("test")

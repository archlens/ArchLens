import libqtile as qtile
from src.core.bt_graph import BTGraph


def update(graph: BTGraph):
    t = graph.get_bt_file("widget.bluetooth")
    graph.change_scope("libqtile.backend")


def settings():
    return {
        "diagram_name": "Qtile Diagram",
        "project": qtile,
    }

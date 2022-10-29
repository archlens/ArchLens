import libqtile as qtile
from src.core.bt_graph import BTGraph


def update(graph: BTGraph):
    t = graph.get_bt_file("widget.bluetooth")


def settings():
    return {
        "diagram_name": "Qtile Diagram",
        "project": qtile,
    }

from src.core.bt_graph import BTGraph
import tp_src as test_project


def update(graph: BTGraph):
    api_file = graph.get_bt_file("tp_src.api.api")
    core_file = graph.get_bt_file("tp_src.tp_core.core")
    api_file.must_depend_on(core_file)

    api_module = graph.get_bt_module("tp_src.api")
    sub_api_module = graph.get_bt_module("tp_src.api.sub_api")
    controller_module = graph.get_bt_module("tp_src.controller")
    core_module = graph.get_bt_module("tp_src.tp_core")

    api_module.cant_depend_on(controller_module)
    sub_api_module.cant_depend_on(core_module)

    graph.change_scope("tp_src.api")


def settings():
    return {
        "diagram_name": "test project",
        "project": test_project,
    }

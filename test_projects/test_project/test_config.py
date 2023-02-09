from src.core.bt_graph import BTGraph
import tp_src as test_project


def update(graph: BTGraph):
    # Get files
    api_file = graph.get_bt_file("tp_src.api.api")
    api_sub_file = graph.get_bt_file("tp_src.api.sub_api.sub_api")
    core_file = graph.get_bt_file("tp_src.tp_core.core")

    # Add file constraints
    api_file.must_depend_on(core_file)
    api_sub_file.must_depend_on(core_file)

    # Get modules
    api_module = graph.get_bt_module("tp_src.api")
    core_module = graph.get_bt_module("tp_src.tp_core")
    controller_module = graph.get_bt_module("tp_src.controller")

    # Add module constraints
    controller_module.cant_depend_on(core_module)
    controller_module.cant_depend_on(api_module)
    core_module.cant_depend_on(api_module)
    
    


def settings():
    return {
        "diagram_name": "test project",
        "project": test_project,
    }

from src.core.bt_graph import BTGraph
import zeeguu
from src.core.bt_file import BTFile


# def setup():
#     app_node = BTFile(code_path="zeeguu.api.app", label="encoding")
#     config_node = BTFile(
#         code_path="zeeguu.core.configuration.configuration", label="config"
#     )

#     app_node.must_depend(config_node)

#     return [app_node, config_node]


def update(graph: BTGraph):
    pass


def settings():
    return {
        "diagram_name": "Zeeguu Diagram",
        "project": zeeguu,
    }

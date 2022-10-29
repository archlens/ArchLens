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
    utils_module = graph.get_bt_module("zeeguu.core.util")
    model_module = graph.get_bt_module("zeeguu.core.model")
    content_retriever_module = graph.get_bt_module("zeeguu.core.content_retriever")

    content_retriever_module.cant_depend_on(utils_module)  # Policy passes
    # model_module.cant_depend_on(utils_module)  # Policy fails

    article_downloader_file = graph.get_bt_file(
        "zeeguu.core.content_retriever.article_downloader"
    )
    converting_from_mysql_file = graph.get_bt_file(
        "zeeguu.core.elastic.converting_from_mysql"
    )
    article_downloader_file.cant_depend_on(converting_from_mysql_file)  # Policy fails


def settings():
    return {
        "diagram_name": "Zeeguu Diagram",
        "project": zeeguu,
    }

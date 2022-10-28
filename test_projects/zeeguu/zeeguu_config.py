import zeeguu
from src.core.bt_file import BTFile


def setup():
    app_node = BTFile(code_path="zeeguu.api.app", label="encoding")
    config_node = BTFile(
        code_path="zeeguu.core.configuration.configuration", label="config"
    )

    app_node.whitelist(config_node)

    return [app_node, config_node]


def settings():
    return {
        "diagram_name": "Zeeguu Diagram",
        "project": zeeguu,
    }

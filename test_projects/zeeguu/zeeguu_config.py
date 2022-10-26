import zeeguu
from src.core import BTNode


def setup():
    app_node = BTNode(connected_code="zeeguu.api.app", label="encoding")
    config_node = BTNode(
        connected_code="zeeguu.core.configuration.configuration", label="config"
    )

    app_node == config_node

    return [app_node, config_node]


def settings():
    return {
        "diagram_name": "Zeeguu Diagram",
        "project": zeeguu,
    }

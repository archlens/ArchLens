# from cmath import cos
# from src.test_project.core import core
# from src.test_project.api import api
# from src.test_project.controller import controller
# from src.test_project.core import core_extra
# from src.test_project.interface import cosmos_interface

# from src.core import BTNode
from src import test_project
from src.zeeguu import zeeguu


def setup():
    # api_node = BTNode(connected_code=api, label="api")
    # core_node = BTNode(connected_code=core, label="core")
    # third_party_api = BTNode(label="Awesome API")
    # cosmos_node = BTNode(label="cosmos", connected_code=cosmos_interface)

    # nodes = [api_node, core_node, third_party_api, cosmos_node]

    # api_node >> third_party_api

    # # api_node != core_node
    # api_node == cosmos_node

    # return nodes
    return []


def settings():
    return {
        "diagram_name": "Zeeguu Diagram",
        "project": zeeguu,
    }

# from test_projects.zeeguu-api import zeeguu
# from test_projects import test_project as zeeguu
from test_projects import zeeguu


def setup():
    return []


def settings():
    return {
        "diagram_name": "Zeeguu Diagram",
        "project": zeeguu,
    }

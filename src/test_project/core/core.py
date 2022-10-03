from test_project.controller.controller import save
from test_project.core.core_extra import test_def


def add(x, y):
    save(x)
    save(y)
    return x + y

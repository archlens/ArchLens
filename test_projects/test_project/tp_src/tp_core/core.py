from tp_src.controller.controller import save

from tp_src.tp_core.tp_core import sub_core_file


def add(x, y):
    save(x)
    save(y)
    return x + y

import astroid
import os

from src.core.bt_file import BTFile


class BTModule:
    parent_module: "BTModule" = None
    child_module: list["BTModule"] = None

    file_list: list["BTFile"] = None

    ast: astroid.Module = None

    def __init__(self, file_path: str) -> None:
        self.ast = astroid.MANAGER.ast_from_file(file_path)
        self.child_module = []
        self.file_list = []

    @property
    def name(self):
        return self.ast.name.split(".")[-1]

    @property
    def path(self):
        return "/".join(self.ast.file.split("/")[:-1])

    def add_files(self):
        files = [
            element
            for element in os.listdir(self.path)
            if element.endswith(".py") and "__" not in element
        ]

        for file in files:
            bt_file = BTFile(label=file.split("/")[-1])
            bt_file.ast = astroid.MANAGER.ast_from_file(os.path.join(self.path, file))
            self.file_list.append(bt_file)

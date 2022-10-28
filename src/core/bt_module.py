import astroid


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

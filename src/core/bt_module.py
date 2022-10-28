class BTModule:
    module_name: str
    module_list: list["BTModule"]
    file_list: list["BTFile"]

    def __init__(self, module_path: str) -> None:
        pass

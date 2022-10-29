class BTFilePolicy:
    def validate(self, edges: list["BTFile"]) -> bool:
        raise Exception("validate not implemented")


class FilePolicyCantDependPolicy(BTFilePolicy):
    cant_depend_node: "BTFile" = None

    def __init__(self, cant_depend_node) -> None:
        super().__init__()
        self.cant_depend_node = cant_depend_node

    def validate(self, edges: list["BTFile"]) -> bool:
        for edge in edges:
            if edge.file == self.cant_depend_node.file:
                return False
        return True


class FilePolicyMustDependPolicy(BTFilePolicy):
    must_depend_node: "BTFile" = None

    def __init__(self, must_depend_node) -> None:
        super().__init__()
        self.must_depend_node = must_depend_node

    def validate(self, edges: list["BTFile"]) -> bool:
        for edge in edges:
            if edge.file == self.must_depend_node.file:
                return True
        return False

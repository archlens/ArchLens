class BTPolicy:
    def validate() -> bool:
        raise Exception("validate not implemented")


class BlacklistPolicy(BTPolicy):
    edges: list["BTNode"] = None
    blacklisted_node: "BTNode" = None

    def __init__(self, edges, blacklisted_node) -> None:
        super().__init__()
        self.edges = edges
        self.blacklisted_node = blacklisted_node

    def validate(self) -> bool:
        for edge in self.edges:
            if edge.file == self.blacklisted_node.file:
                return False
        return True


class WhitelistPolicy(BTPolicy):
    edges: list["BTNode"] = None
    whitelisted_node: "BTNode" = None

    def __init__(self, edges, whitelisted_node) -> None:
        super().__init__()
        self.edges = edges
        self.whitelisted_node = whitelisted_node

    def validate(self) -> bool:
        for edge in self.edges:
            if edge.file == self.whitelisted_node.file:
                return True
        return False

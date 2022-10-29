class BTPolicy:
    def validate(edges: list["BTFile"]) -> bool:
        raise Exception("validate not implemented")


class BlacklistPolicy(BTPolicy):
    blacklisted_node: "BTFile" = None

    def __init__(self, blacklisted_node) -> None:
        super().__init__()
        self.blacklisted_node = blacklisted_node

    def validate(self, edges: list["BTFile"]) -> bool:
        for edge in edges:
            if edge.file == self.blacklisted_node.file:
                return False
        return True


class WhitelistPolicy(BTPolicy):
    whitelisted_node: "BTFile" = None

    def __init__(self, whitelisted_node) -> None:
        super().__init__()
        self.whitelisted_node = whitelisted_node

    def validate(self, edges: list["BTFile"]) -> bool:
        for edge in edges:
            if edge.file == self.whitelisted_node.file:
                return True
        return False

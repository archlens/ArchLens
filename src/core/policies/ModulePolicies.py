class BTModulePolicy:
    def validate(self) -> bool:
        raise Exception("validate not implemented")


class ModulePolicyCantDepend(BTModulePolicy):
    cant_depend_module: "BTModule" = None

    def __init__(self, other) -> None:
        self.cant_depend_module = other
        super().__init__()

    def validate(self, module: "BTModule") -> bool:
        # TODO check that no file is dependent on file in other module
        super().validate()

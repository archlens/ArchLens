class BTModulePolicy:
    def validate(self) -> bool:
        raise Exception("validate not implemented")


class ModulePolicyCantDepend(BTModulePolicy):
    cant_depend_module: "BTModule" = None
    module: "BTModule" = None

    def __init__(self, other, module: "BTModule") -> None:
        self.cant_depend_module = other
        self.module = module

    def validate(self) -> bool:
        other_file_set = set(
            [other.file for other in self.cant_depend_module.get_files_recursive()]
        )
        for module_files in self.module.get_files_recursive():
            for edge in module_files.edge_to:
                parent_modules = edge.module.get_parent_module_recursive()
                parent_modules.append(edge.module)
                if self.cant_depend_module in parent_modules:
                    return False
        return True

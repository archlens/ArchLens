from src.core.bt_module import BTModule
from src.plantumlv2.utils import get_pu_package_path_from_bt_package
from enum import Enum

from src.utils.config_manager_singleton import ConfigManagerSingleton


class EntityState(str, Enum):
    DELETED = "#Red"
    NEUTRAL = ""
    CREATED = "#Green"


PACKAGE_NAME_SPLITTER = "."


class PuPackage:
    name = ""
    parent: "PuPackage" = None
    sub_modules: list["PuPackage"] = None
    state: EntityState = EntityState.NEUTRAL
    pu_dependency_list: list["PuDependency"] = None
    bt_package: BTModule = None

    def __init__(self, bt_package: BTModule) -> None:
        self.pu_dependency_list = []
        self.name = PACKAGE_NAME_SPLITTER.join(
            get_pu_package_path_from_bt_package(bt_package).split("/")
        )
        self.bt_package = bt_package
        self.sub_modules = []

    @property
    def path(self):
        return get_pu_package_path_from_bt_package(self.bt_package)

    @property
    def parent_path(self):
        return get_pu_package_path_from_bt_package(
            self.bt_package.parent_module
        )

    def setup_dependencies(self, pu_package_map: dict[str, "PuPackage"]):
        bt_dependencies = self.bt_package.get_module_dependencies()
        self.parent = pu_package_map.get(self.parent_path)
        if self.parent:
            self.parent.sub_modules.append(self)
        for bt_package_dependency in bt_dependencies:
            pu_path = get_pu_package_path_from_bt_package(
                bt_package_dependency
            )
            if pu_path == self.path:
                continue
            try:
                pu_package_dependency = pu_package_map[pu_path]
                self.pu_dependency_list.append(
                    PuDependency(
                        self,
                        pu_package_dependency,
                        self.bt_package,
                        bt_package_dependency,
                    )
                )
            except Exception:
                pass

    def render_package(self) -> str:
        config_manager = ConfigManagerSingleton()
        state_str = self.state.value
        if self.state == EntityState.NEUTRAL:
            state_str = config_manager.package_color

        return f'package "{self.name}" {state_str}'

    def render_dependency(self) -> str:
        return "\n".join(
            [
                pu_dependency.render()
                for pu_dependency in self.pu_dependency_list
            ]
        )

    def filter_excess_packages_dependencies(
        self, used_packages: set["PuPackage"]
    ):
        for sub_module in self.sub_modules:
            if sub_module not in used_packages:
                sub_module.filter_excess_packages_dependencies(used_packages)
                self.pu_dependency_list.extend(sub_module.pu_dependency_list)

        # change all from package to self
        for dependency in self.pu_dependency_list:
            dependency.from_package = self

        # filter all dependencies pointing to self
        self.pu_dependency_list = [
            dependency
            for dependency in self.pu_dependency_list
            if dependency.to_package != self
            and dependency.to_package in used_packages
        ]

        # Makes sure that there is only one dependency per package
        aggregate_dependency_map: dict[str, PuDependency] = {}
        for dependency in self.pu_dependency_list:
            if dependency.id not in aggregate_dependency_map:
                aggregate_dependency_map[dependency.id] = dependency
            else:
                dep = aggregate_dependency_map[dependency.id]
                dep.dependency_count += dependency.dependency_count
        self.pu_dependency_list = list(aggregate_dependency_map.values())

    def get_dependency_map(self) -> dict[str, "PuDependency"]:
        return {
            dependency.to_package.path: dependency
            for dependency in self.pu_dependency_list
        }


class PuDependency:
    state: EntityState = EntityState.NEUTRAL

    from_package: PuPackage = None
    to_package: PuPackage = None

    from_bt_package: BTModule = None
    to_bt_package: BTModule = None

    dependency_count = 0

    def __init__(
        self,
        from_package: PuPackage,
        to_package: PuPackage,
        from_bt_package: BTModule,
        to_bt_package: BTModule,
    ) -> None:
        self.from_package = from_package
        self.to_package = to_package
        self.from_bt_package = from_bt_package
        self.to_bt_package = to_bt_package
        self.dependency_count = self.from_bt_package.get_dependency_count(
            self.to_bt_package
        )

    @property
    def id(self):
        return f"{self.from_package.name}-->{self.to_package.name}"

    def render(self) -> str:
        config_manager = ConfigManagerSingleton()
        dependency_count_str = ""
        if config_manager.show_dependency_count:
            dependency_count_str = f": {self.dependency_count}"
        from_name = self.from_package.name
        to_name = self.to_package.name
        return f'"{from_name}"-->"{to_name}" {self.state.value} {dependency_count_str}'

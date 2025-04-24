from src.core.bt_module import BTModule, BTFile
from src.views.utils import get_view_package_path_from_bt_package
from enum import Enum

from src.utils.config_manager_singleton import ConfigManagerSingleton


class EntityState(str, Enum):
    DELETED = "#Red"
    NEUTRAL = ""
    CREATED = "#Green"


PACKAGE_NAME_SPLITTER = "."


class ViewPackage:
    name = ""
    parent: "ViewPackage" = None
    sub_modules: list["ViewPackage"] = None
    state: EntityState = EntityState.NEUTRAL
    view_dependency_list: list["ViewDependancy"] = None
    bt_package: BTModule = None

    def __init__(self, bt_package: BTModule) -> None:
        self.view_dependency_list = []
        self.name = PACKAGE_NAME_SPLITTER.join(
            get_view_package_path_from_bt_package(bt_package).split("/")
        )
        self.bt_package = bt_package
        self.sub_modules = []

    @property
    def path(self):
        return get_view_package_path_from_bt_package(self.bt_package)

    @property
    def parent_path(self):
        return get_view_package_path_from_bt_package(self.bt_package.parent_module)

    def setup_dependencies(self, view_package_map: dict[str, "ViewPackage"]):
        bt_dependencies = self.bt_package.get_module_dependencies()
        self.parent = view_package_map.get(self.parent_path)
        if self.parent:
            self.parent.sub_modules.append(self)
        for bt_package_dependency in bt_dependencies:
            view_path = get_view_package_path_from_bt_package(bt_package_dependency)
            if view_path == self.path:
                continue
            try:
                view_package_dependency = view_package_map[view_path]
                self.view_dependency_list.append(
                    ViewDependancy(
                        self,
                        view_package_dependency,
                        self.bt_package,
                        bt_package_dependency,
                    )
                )
            except Exception:
                pass

    def get_parent_list(self):
        res = []
        if self.parent is None:
            return res
        res.append(self.parent)
        res.extend(self.parent.get_parent_list())
        return res

    def render_package_pu(self) -> str:
        config_manager = ConfigManagerSingleton()
        state_str = self.state.value
        if self.state == EntityState.NEUTRAL:
            state_str = config_manager.package_color

        return f'package "{self.name}" {state_str}'

    def render_dependency_pu(self) -> str:
        return "\n".join(
            [
                view_dependency.render_pu()
                for view_dependency in self.view_dependency_list
            ]
        )

    def render_package_json(self):
        return {"name": self.name, "state": self.state.name}

    def render_dependency_json(self):
        return [
            view_dependency.render_json()
            for view_dependency in self.view_dependency_list
        ]

    def filter_excess_packages_dependencies(self, used_packages: set["ViewPackage"]):
        for sub_module in self.sub_modules:
            if sub_module not in used_packages:
                sub_module.filter_excess_packages_dependencies(used_packages)
                self.view_dependency_list.extend(sub_module.view_dependency_list)

        for dependency in self.view_dependency_list:
            parent_list = dependency.to_package.get_parent_list()
            for parent in parent_list:
                if parent in used_packages:
                    dep = ViewDependancy(
                        self,
                        parent,
                        dependency.from_bt_package,
                        dependency.to_bt_package,
                    )
                    self.view_dependency_list.append(dep)
                    break

        # change all from package to self
        for dependency in self.view_dependency_list:
            dependency.from_package = self

        # filter all dependencies pointing to self
        self.view_dependency_list = [
            dependency
            for dependency in self.view_dependency_list
            if dependency.to_package != self and dependency.to_package in used_packages
        ]

        # Makes sure that there is only one dependency per package
        aggregate_dependency_map: dict[str, ViewDependancy] = {}
        for dependency in self.view_dependency_list:
            if dependency.id not in aggregate_dependency_map:
                aggregate_dependency_map[dependency.id] = dependency
            else:
                dep = aggregate_dependency_map[dependency.id]
                dep.dependency_count += dependency.dependency_count
                dep.edge_files.extend(dependency.edge_files)

        self.view_dependency_list = list(aggregate_dependency_map.values())

    def get_dependency_map(self) -> dict[str, "ViewDependancy"]:
        return {
            dependency.to_package.path: dependency
            for dependency in self.view_dependency_list
        }


class ViewDependancy:
    state: EntityState = EntityState.NEUTRAL

    from_package: ViewPackage = None
    to_package: ViewPackage = None

    from_bt_package: BTModule = None
    to_bt_package: BTModule = None

    edge_files: list[tuple[BTFile, BTFile]] = []

    dependency_count = 0

    render_diff = {}

    def __init__(
        self,
        from_package: ViewPackage,
        to_package: ViewPackage,
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
        self.edge_files = self.from_bt_package.get_file_level_relations(
            self.to_bt_package
        )

    @property
    def id(self):
        return f"{self.from_package.name}-->{self.to_package.name}"

    def render_pu(self) -> str:
        config_manager = ConfigManagerSingleton()

        if not self.render_diff:
            dependency_count_str = ""
            if config_manager.show_dependency_count:
                dependency_count_str = f": {self.dependency_count}"
            from_name = self.from_package.name
            to_name = self.to_package.name
            return (
                f'"{from_name}"-->"{to_name}" {self.state.value} {dependency_count_str}'
            )
        else:
            return f'"{self.render_diff["from_package"].name}"-->"{self.render_diff["to_package"].name}" {self.render_diff["color"].value} : {self.render_diff["label"]}'

    def render_json(self) -> dict:
        config_manager = ConfigManagerSingleton()

        if not self.render_diff:
            label = ""
            if config_manager.show_dependency_count:
                label = f"{self.dependency_count}"
            from_package: ViewPackage = self.from_package.name
            to_package: ViewPackage = self.to_package.name
            label = str(self.dependency_count)
            state: EntityState = self.state
            state_str = state.name
        else:
            from_package: ViewPackage = self.render_diff["from_package"].name
            to_package: ViewPackage = self.render_diff["to_package"].name
            label = self.render_diff["label"]
            state: EntityState = self.render_diff["color"]
            state_str = state.name
        return {
            "state": state_str,
            "fromPackage": from_package,
            "toPackage": to_package,
            "label": label,
            "relations": [
                {
                    "from_file": {"name": relation[0].label, "path": relation[0].file},
                    "to_file": {"name": relation[1].label, "path": relation[1].file},
                }
                for relation in self.edge_files
            ],
        }

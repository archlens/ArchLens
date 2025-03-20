class ConfigManagerSingleton:
    _instance = None

    show_dependency_count = None
    package_color = None

    def __new__(cls):
        if cls._instance is None:
            cls._instance = super().__new__(cls)
        return cls._instance

    def setup(self, config: dict):
        self.show_dependency_count = config.get("showDependencyCount", True)
        self.package_color = config.get("packageColor", "#Azure")

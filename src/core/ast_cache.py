from typing import Dict

import astroid
from astroid.manager import AstroidManager


class AstCache:
    cache: Dict[str, astroid.Module]
    am: AstroidManager

    def __init__(self, am: AstroidManager):
        self.cache = {}
        self.am = am

    def get_module(self, module_name: str, root_path):

        if module_name in self.cache:
            return self.cache[module_name]
        else:
            module = self.am.ast_from_module_name(
                module_name,
                context_file=root_path,
            )

            self.cache[module_name] = module

            return module
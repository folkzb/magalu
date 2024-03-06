from typing import List
from spec_types import SpecTranformer
from oapi_types import OAPI
import re


class RemoveComponentTransformer(SpecTranformer):
    def __init__(self, pattern: str):
        self.pattern = re.compile(pattern)

    def transform(self, oapi: OAPI):
        self._remove(oapi)

    def _remove(self, oapi: OAPI):
        spec = oapi.obj

        to_delete: List[str] = []
        for component in list(spec.get("components", {}).get("schemas", {})):
            if self.pattern.match(component):
                to_delete.append(component)

        for component in to_delete:
            del spec.get("components", {}).get("schemas", {})[component]

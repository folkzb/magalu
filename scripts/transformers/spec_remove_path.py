from typing import List
from spec_types import SpecTranformer
from oapi_types import OAPI
import re


class RemovePathTransformer(SpecTranformer):
    def __init__(self, pattern: str):
        self.pattern = re.compile(pattern)

    def transform(self, oapi: OAPI):
        self._remove(oapi)

    def _remove(self, oapi: OAPI):
        spec = oapi.obj

        to_remove: List[str] = []
        for path in spec.get("paths", {}).keys():
            if self.pattern.match(path):
                to_remove.append(path)

        for path in to_remove:
            del spec.get("paths", {})[path]

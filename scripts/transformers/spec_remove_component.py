from spec_types import SpecTranformer
from oapi_types import OAPI
import re


class RemoveComponentTransformer(SpecTranformer):
    def __init__(self, pattern: str):
        self.pattern = pattern

    def transform(self, oapi: OAPI):
        self._remove(oapi)

    def _remove(self, oapi: OAPI):
        spec = oapi.obj
        for component in list(spec.get("components", {}).get("schemas", {})):
            if re.search(self.pattern, component, re.I) is not None:
                del spec["components"]["schemas"][component]

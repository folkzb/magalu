from spec_types import OAPISchema, SpecTranformer
import re


class RemoveComponentTransformer(SpecTranformer):
    def __init__(self, pattern: str):
        self.pattern = pattern

    def transform(self, spec: OAPISchema) -> OAPISchema:
        return self._remove(spec)

    def _remove(self, spec: OAPISchema) -> OAPISchema:
        for component in list(spec["components"]["schemas"]):
            if re.search(self.pattern, component, re.I) is not None:
                del spec["components"]["schemas"][component]
        return spec

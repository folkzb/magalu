from spec_types import SpecTranformer
from oapi_types import OAPI, OAPIObject
import re


class RemovePathTransformer(SpecTranformer):
    def __init__(self, pattern: str):
        self.pattern = pattern

    def transform(self, oapi: OAPI):
        self._remove(oapi)

    def _remove(self, oapi: OAPI):
        spec = oapi.obj
        paths_to_remove = self._remove_from_field(spec, "paths")
        if paths_to_remove is not None:
            self._remove_from_spec(spec, paths_to_remove)

    def _remove_from_field(self, spec: OAPIObject, oapi_field_name: str):
        refs = []
        obj = {}
        for path in spec.get(oapi_field_name, []):
            if re.search(self.pattern, path, re.I) is not None:
                refs.append(path)
        if refs:
            obj[oapi_field_name] = refs

    def _remove_from_spec(self, spec: OAPIObject, refs: dict):
        for keys in refs:
            for value in refs[keys]:
                del spec[keys][value]

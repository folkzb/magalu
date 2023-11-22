from spec_types import OAPISchema, SpecTranformer
import re


class RemovePathTransformer(SpecTranformer):
    def __init__(self, pattern: str):
        self.pattern = pattern

    def transform(self, spec: OAPISchema) -> OAPISchema:
        return self._remove(spec)

    def _remove(self, spec: OAPISchema) -> OAPISchema:
        paths_to_remove = self._remove_from_field(spec, "paths")
        if paths_to_remove is not None:
            self._remove_from_spec(spec, paths_to_remove)
        return spec

    def _remove_from_field(self, spec: OAPISchema, oapi_field_name: str):
        refs = []
        obj = {}
        for path in spec.get(oapi_field_name, []):
            if re.search(self.pattern, path, re.I) is not None:
                refs.append(path)
        if refs:
            obj[oapi_field_name] = refs
        return obj

    def _remove_from_spec(self, spec: OAPISchema, refs: dict):
        for keys in refs:
            for value in refs[keys]:
                del spec[keys][value]

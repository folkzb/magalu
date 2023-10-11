from typing import Any, List, Tuple

from spec_types import OAPISchema, SpecTranformer


class RemoveParamTransformer(SpecTranformer):
    def __init__(self, param_name: str):
        self.param_name = param_name

    def transform(self, spec: OAPISchema) -> OAPISchema:
        return self.__remove_param(spec, param_name="x-tenant-id")

    def __remove_param(self, spec: OAPISchema, param_name: str) -> OAPISchema:
        refs_for_removal = set()
        for path in spec.get("paths", {}).values():
            for action in path.values():
                if not isinstance(action, dict) or "parameters" not in action:
                    continue

                filtered_params, removable_refs = self.__filter_params_and_refs(
                    action.get("parameters", [{}]), spec, param_name
                )
                refs_for_removal.update(removable_refs)

                if not filtered_params:
                    del action["parameters"]
                else:
                    action["parameters"] = filtered_params

        self.__remove_param_refs(spec, refs_for_removal)
        return spec

    def __filter_params_and_refs(
        self, params: List[str], spec: OAPISchema, param_name: str
    ) -> Tuple[List[str], List[str]]:
        refs = []
        filtered_params = []
        for p in params:
            pv = p
            ref = None

            if "$ref" in pv:
                ref = pv.get("$ref")
                ref_path = ref.removeprefix("#/").split("/")

                pv = spec
                for rp in ref_path:
                    pv = pv[rp]

            if pv.get("name") != param_name:
                filtered_params.append(p)
            elif ref is not None:
                refs.append(ref)
                ref = None

        return filtered_params, refs

    def __remove_param_refs(self, spec: OAPISchema, refs: List[str]):
        def should_delete(value: Any, keys: List[str]):
            if len(keys) == 0:
                return True
            else:
                if should_delete(value[keys[0]], keys[1:]):
                    del value[keys[0]]

                return len(value) == 0

        for ref in refs:
            ref_path = ref.removeprefix("#/").split("/")
            if should_delete(spec, ref_path):
                del spec[ref_path[0]]

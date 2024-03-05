from typing import Any, List, Tuple

from spec_types import SpecTranformer
from oapi_types import OAPI, OAPIObject


class RemoveParamTransformer(SpecTranformer):
    def __init__(self, param_name: str):
        self.param_name = param_name

    def transform(self, oapi: OAPI):
        self._remove_param(oapi, param_name=self.param_name)

    def _remove_param(self, oapi: OAPI, param_name: str):
        spec = oapi.obj
        refs_for_removal = set()
        for path in spec.get("paths", {}).values():
            for action in path.values():
                if not isinstance(action, dict) or "parameters" not in action:
                    continue

                filtered_params, removable_refs = self._filter_params_and_refs(
                    action.get("parameters", [{}]), spec, param_name
                )
                refs_for_removal.update(removable_refs)

                if not filtered_params:
                    del action["parameters"]
                else:
                    action["parameters"] = filtered_params

        self._remove_param_refs(spec, list(refs_for_removal))

    def _filter_params_and_refs(
        self, params: List[str], spec: OAPIObject, param_name: str
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

    def _remove_param_refs(self, spec: OAPIObject, refs: List[str]):
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

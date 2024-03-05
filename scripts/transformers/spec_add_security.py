from spec_types import SpecTranformer
from oapi_types import OAPI
from typing import Dict, List

# TODO: Add other operation methods?
read_requirements = ["get"]
write_requirements = ["post", "patch", "delete"]


class AddSecurityTransformer(SpecTranformer):
    def __init__(self, product_name: str):
        self.product_name = product_name

    def _get_security_schema(
        self, http_method: str
    ) -> List[Dict[str, List[str]]] | None:
        scope = ""
        if http_method in read_requirements:
            scope = "read"
        elif http_method in write_requirements:
            scope = "write"
        else:
            return None

        return [{"OAuth2": [self.product_name + "." + scope]}]

    def transform(self, oapi: OAPI):
        """
        Assume all operations need security and add them, with scope using
        product_name
        """
        spec = oapi.obj
        paths = spec.get("paths", {})
        for operations in paths.values():
            for http_method, op in operations.items():
                if op.get("security") is not None:
                    continue

                security = self._get_security_schema(http_method)
                if security is None:
                    continue

                op["security"] = security

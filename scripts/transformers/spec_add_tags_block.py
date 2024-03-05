from spec_types import SpecTranformer
from oapi_types import OAPI, OAPIOperationObject
from typing import Dict


class AddTagsBlockTransformer(SpecTranformer):
    def transform(self, oapi: OAPI):
        """
        When a spec is missing the 'tags' block, run through all operations
        collecting the tags and create the block with all of them
        """

        spec = oapi.obj
        if "tags" in spec:
            return

        tags: Dict[str, str] = {}

        for path_item in spec.get("paths", {}).values():
            path_ops: Dict[str, OAPIOperationObject | None] = {
                "get": path_item.get("get"),
                "post": path_item.get("post"),
                "put": path_item.get("put"),
                "patch": path_item.get("patch"),
                "delete": path_item.get("delete"),
            }

            for op in path_ops.values():
                if op is None:
                    continue

                for tag in op.get("tags", []):
                    tags[tag] = tag

        spec["tags"] = []
        for name in sorted(tags.keys()):
            spec["tags"].append({"name": name, "description": tags[name]})

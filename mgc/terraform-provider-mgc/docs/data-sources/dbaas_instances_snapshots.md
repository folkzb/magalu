---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "mgc_dbaas_instances_snapshots Data Source - terraform-provider-mgc"
subcategory: "Database"
description: |-
  List all snapshots for a database instance.
---

# mgc_dbaas_instances_snapshots (Data Source)

List all snapshots for a database instance.

## Example Usage

```terraform
data "mgc_dbaas_instances_snapshots" "all" {
  instance_id = "instance-123"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `instance_id` (String) ID of the instance

### Read-Only

- `snapshots` (Attributes List) List of snapshots (see [below for nested schema](#nestedatt--snapshots))

<a id="nestedatt--snapshots"></a>
### Nested Schema for `snapshots`

Required:

- `instance_id` (String) ID of the instance

Read-Only:

- `created_at` (String) Creation timestamp
- `description` (String) Description of the snapshot
- `id` (String) ID of the snapshot
- `name` (String) Name of the snapshot
- `size` (Number) Size of the snapshot in bytes
- `status` (String) Status of the snapshot
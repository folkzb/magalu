---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "mgc_object_storage_bucket Data Source - terraform-provider-mgc"
subcategory: ""
description: |-
  Get details of bucket.
---

# mgc_object_storage_bucket (Data Source)

Get details of bucket.



<!-- schema generated by tfplugindocs -->
## Schema

### Read-Only

- `buckets` (Attributes List) List of ssh-keys. (see [below for nested schema](#nestedatt--buckets))

<a id="nestedatt--buckets"></a>
### Nested Schema for `buckets`

Required:

- `name` (String) Bucket name

Read-Only:

- `grantee` (Attributes List) Bucket grantee (see [below for nested schema](#nestedatt--buckets--grantee))
- `mfadelete` (String) MFA Delete
- `owner` (Attributes List) Bucket owner (see [below for nested schema](#nestedatt--buckets--owner))
- `versioning` (String) Versioning status

<a id="nestedatt--buckets--grantee"></a>
### Nested Schema for `buckets.grantee`

Read-Only:

- `display_name` (String) Grantee Name
- `id` (String) Grantee ID
- `permission` (String) Grantee permission


<a id="nestedatt--buckets--owner"></a>
### Nested Schema for `buckets.owner`

Read-Only:

- `display_name` (String) Owner Name
- `id` (String) Owner ID
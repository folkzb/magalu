---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "mgc_network_vpcs_subnet Data Source - terraform-provider-mgc"
subcategory: "Network"
description: |-
  Network VPC Subnet
---

# mgc_network_vpcs_subnet (Data Source)

Network VPC Subnet

## Example Usage

```terraform
data "mgc_network_vpcs_subnet" "example" {
  id = mgc_network_vpcs_subnets.example.id
}

output "datasource_subnet_id" {
  value = data.mgc_network_vpcs_subnet.example
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `id` (String) The ID of the subnet

### Read-Only

- `cidr_block` (String) The CIDR block of the subnet
- `description` (String) The description of the subnet
- `dhcp_pools` (Attributes List) The DHCP pools of the subnet (see [below for nested schema](#nestedatt--dhcp_pools))
- `dns_nameservers` (List of String) The DNS nameservers of the subnet
- `gateway_ip` (String) The gateway IP of the subnet
- `ip_version` (String) The IP version of the subnet
- `name` (String) The name of the subnet
- `updated` (String) The updated timestamp of the subnet
- `vpc_id` (String) The VPC ID of the subnet
- `zone` (String) The zone of the subnet

<a id="nestedatt--dhcp_pools"></a>
### Nested Schema for `dhcp_pools`

Read-Only:

- `end` (String) The end of the DHCP pool
- `start` (String) The start of the DHCP pool
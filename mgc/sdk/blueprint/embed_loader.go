// Code generated by blueprint_index_gen. DO NOT EDIT.

//go:build embed

//nolint

package blueprint

import (
	"os"
	"syscall"
	"magalu.cloud/core/dataloader"
)

type embedLoader map[string][]byte

func GetEmbedLoader() dataloader.Loader {
	return embedLoaderInstance
}

func (f embedLoader) Load(name string) ([]byte, error) {
	if data, ok := embedLoaderInstance[name]; ok {
		return data, nil
	}
	return nil, &os.PathError{Op: "open", Path: name, Err: syscall.ENOENT}
}

func (f embedLoader) String() string {
	return "embedLoader"
}

var embedLoaderInstance = embedLoader{
	"index.blueprint.yaml": ([]byte)("{\"modules\":[{\"description\":\"Operations for Block Storage API\",\"name\":\"block-storage\",\"path\":\"block-storage.blueprint.yaml\",\"url\":\"https://block-storage.magalu.cloud\",\"version\":\"1.52.0\"},{\"description\":\"Operations for Network API\",\"name\":\"network\",\"path\":\"network.blueprint.yaml\",\"url\":\"https://network.magalu.cloud\",\"version\":\"1.99.5\"}],\"version\":\"1.0.0\"}"),
	"block-storage.blueprint.yaml": ([]byte)("{\"blueprint\":\"1.0.0\",\"children\":[{\"children\":[{\"configsSchema\":{\"$ref\":\"blueprint#/components/configsSchemas/default\"},\"description\":\"Attach a volume to a virtual machine instance\",\"links\":{\"get\":{\"description\":\"Check if this Public IP is attached to a Port\",\"parameters\":{\"id\":\"$.parameters.id\"},\"target\":\"/block-storage/volumes/get\",\"waitTermination\":{\"errorJsonPathQuery\":\"$.result.status == \\\"attaching_error\\\" || (!hasKey($.result, \\\"attachment\\\") && $.result.status == \\\"completed\\\")\\n\",\"interval\":\"5s\",\"jsonPathQuery\":\"hasKey($.result, \\\"attachment\\\") && $.result.attachment.machine_id == $.owner.parameters.virtual_machine_id\\n\",\"maxRetries\":5}}},\"name\":\"create\",\"parametersSchema\":{\"$ref\":\"blueprint#/components/schemas/VolumeAttachObject\"},\"resultSchema\":{\"$ref\":\"blueprint#/components/schemas/VolumeAttachObject\"},\"scopes\":[\"block-storage.write\"],\"steps\":[{\"parameters\":{\"id\":\"$.parameters.block_storage_id\",\"virtual_machine_id\":\"$.parameters.virtual_machine_id\"},\"target\":\"/block-storage/volumes/attach\"}]},{\"configsSchema\":{\"$ref\":\"blueprint#/components/configsSchemas/default\"},\"description\":\"Check if a volume is attached to a virtual machine instance\",\"name\":\"get\",\"parametersSchema\":{\"$ref\":\"blueprint#/components/schemas/VolumeAttachObject\"},\"result\":\"{\\n  \\\"block_storage_id\\\": $.parameters.block_storage_id,\\n  \\\"virtual_machine_id\\\": $.parameters.virtual_machine_id\\n}\\n\",\"resultSchema\":{\"$ref\":\"blueprint#/components/schemas/VolumeAttachObject\"},\"scopes\":[\"block-storage.read\"],\"steps\":[{\"parameters\":{\"id\":\"$.parameters.block_storage_id\"},\"target\":\"/block-storage/volumes/get\"}],\"waitTermination\":{\"errorJsonPathQuery\":\"$.last.result.status == \\\"attaching_error\\\" || (!hasKey($.last.result, \\\"attachment\\\") && $.last.result.status == \\\"completed\\\")\\n\",\"interval\":\"5s\",\"jsonPathQuery\":\"hasKey($.last.result, \\\"attachment\\\") && $.last.result.attachment.machine_id == $.parameters.virtual_machine_id\\n\",\"maxRetries\":5}},{\"configsSchema\":{\"$ref\":\"blueprint#/components/configsSchemas/default\"},\"confirm\":\"Detaching {{ .parameters.block_storage_id }} cannot be undone.\\nConfirm?\\n\",\"description\":\"Detach a volume from a virtual machine instance\",\"name\":\"delete\",\"parametersSchema\":{\"$ref\":\"blueprint#/components/schemas/VolumeAttachObject\"},\"result\":\"{\\n  \\\"block_storage_id\\\": $.parameters.block_storage_id,\\n}\\n\",\"resultSchema\":{\"$ref\":\"blueprint#/components/schemas/VolumeAttachObject\"},\"scopes\":[\"block-storage.write\"],\"steps\":[{\"parameters\":{\"id\":\"$.parameters.block_storage_id\"},\"target\":\"/block-storage/volumes/detach\"}]}],\"description\":\"Block Storage Volume Attachment\",\"isInternal\":true,\"name\":\"volume-attachment\"}],\"components\":{\"configsSchemas\":{\"default\":{\"$ref\":\"/block-storage/volumes/get/configsSchema\"}},\"schemas\":{\"VolumeAttachObject\":{\"properties\":{\"block_storage_id\":{\"$ref\":\"/block-storage/volumes/attach/parametersSchema/properties/id\"},\"virtual_machine_id\":{\"$ref\":\"/block-storage/volumes/attach/parametersSchema/properties/virtual_machine_id\"}},\"type\":\"object\"}}},\"description\":\"Operations for Block Storage API\",\"name\":\"block-storage\",\"url\":\"https://block-storage.magalu.cloud\",\"version\":\"1.52.0\"}"),
	"network.blueprint.yaml": ([]byte)("{\"blueprint\":\"1.0.0\",\"children\":[{\"children\":[{\"$ref\":\"http://magalu.cloud/sdk#/network/vpcs/ports/create\",\"isInternal\":false},{\"children\":[{\"configsSchema\":{\"$ref\":\"blueprint#/components/configsSchemas/default\"},\"description\":\"Attach a Security Group to a Port\",\"links\":{\"get\":{\"description\":\"Check if this Security Group is attached to this Port\",\"parameters\":{\"port_id\":\"$.parameters.port_id\",\"security_group_id\":\"$.parameters.security_group_id\"},\"target\":\"/network/ports/security-group-attachment/get\"}},\"name\":\"create\",\"parametersSchema\":{\"$ref\":\"blueprint#/components/schemas/PortAttachObject\"},\"resultSchema\":{\"$ref\":\"blueprint#/components/schemas/PortAttachObject\"},\"scopes\":[\"network.write\"],\"steps\":[{\"parameters\":{\"port_id\":\"$.parameters.port_id\",\"security_group_id\":\"$.parameters.security_group_id\"},\"target\":\"/network/ports/attach\"},{\"check\":{\"errorMessageTemplate\":\"Security Group {{ .parameters.security_group_id }} was not attached to Port {{ .parameters.port_id }}\\n\",\"jsonPathQuery\":\"hasKey($.current.result, \\\"security_groups\\\") && $.current.result.security_groups[?(@ == $.parameters.security_group_id)].length > 0\\n\"},\"parameters\":{\"port_id\":\"$.parameters.port_id\"},\"target\":\"/network/ports/get\"}]},{\"configsSchema\":{\"$ref\":\"blueprint#/components/configsSchemas/default\"},\"description\":\"Check if a Public IP is attached to a Port\",\"name\":\"get\",\"parametersSchema\":{\"$ref\":\"blueprint#/components/schemas/PortAttachObject\"},\"resultSchema\":{\"$ref\":\"blueprint#/components/schemas/PortAttachObject\"},\"scopes\":[\"network.read\"],\"steps\":[{\"check\":{\"errorMessageTemplate\":\"Security Group {{ .parameters.security_group_id }} is not attached to Port {{ .parameters.port_id }}\\n\",\"jsonPathQuery\":\"hasKey($.current.result, \\\"security_groups\\\") && $.current.result.security_groups[?(@ == $.parameters.security_group_id)].length > 0\\n\"},\"parameters\":{\"port_id\":\"$.parameters.port_id\"},\"retryUntil\":{\"interval\":\"5s\",\"jsonPathQuery\":\"hasKey($.current.result, \\\"security_groups\\\") && $.current.result.security_groups[?(@ == $.parameters.security_group_id)].length > 0\\n\",\"maxRetries\":3},\"target\":\"/network/ports/get\"}]},{\"configsSchema\":{\"$ref\":\"blueprint#/components/configsSchemas/default\"},\"confirm\":\"Detaching Security Group {{ .parameters.security_group_id }} from Port {{ .parameters.port_id }}. Confirm?\\n\",\"description\":\"Detach a Security Group from a Port\",\"name\":\"delete\",\"parametersSchema\":{\"$ref\":\"blueprint#/components/schemas/PortAttachObject\"},\"result\":\"{\\n  \\\"security_group_id\\\": $.parameters.security_group_id,\\n  \\\"port_id\\\": $.parameters.port_id,\\n}\\n\",\"resultSchema\":{\"$ref\":\"blueprint#/components/schemas/PortAttachObject\"},\"scopes\":[\"network.write\"],\"steps\":[{\"parameters\":{\"port_id\":\"$.parameters.port_id\",\"security_group_id\":\"$.parameters.security_group_id\"},\"target\":\"/network/ports/detach\"}]}],\"description\":\"Manage the attachment between a Security Group and a Port\",\"isInternal\":true,\"name\":\"security-group-attachment\"}],\"description\":\"VPC Port\",\"name\":\"ports\"},{\"children\":[{\"$ref\":\"http://magalu.cloud/sdk#/network/vpcs/public-ips/create\",\"isInternal\":false},{\"children\":[{\"configsSchema\":{\"$ref\":\"blueprint#/components/configsSchemas/default\"},\"description\":\"Attach a Public IP to a Port\",\"links\":{\"get\":{\"description\":\"Check if this Public IP is attached to a Port\",\"parameters\":{\"port_id\":\"$.parameters.port_id\",\"public_ip_id\":\"$.parameters.public_ip_id\"},\"target\":\"/network/public_ips/port-attachment/get\"}},\"name\":\"create\",\"parametersSchema\":{\"$ref\":\"blueprint#/components/schemas/PublicIPAttachObject\"},\"resultSchema\":{\"$ref\":\"blueprint#/components/schemas/PublicIPAttachObject\"},\"scopes\":[\"network.write\"],\"steps\":[{\"parameters\":{\"port_id\":\"$.parameters.port_id\",\"public_ip_id\":\"$.parameters.public_ip_id\"},\"target\":\"/network/public_ips/attach\"},{\"check\":{\"errorMessageTemplate\":\"PublicIP {{ .parameters.public_ip_id }} was not attached to Port {{ .parameters.port_id }}\\n\",\"jsonPathQuery\":\"hasKey($.current.result, \\\"port_id\\\") && $.current.result.port_id == $.current.parameters.port_id\\n\"},\"parameters\":{\"public_ip_id\":\"$.parameters.public_ip_id\"},\"target\":\"/network/public_ips/get\"}]},{\"configsSchema\":{\"$ref\":\"blueprint#/components/configsSchemas/default\"},\"description\":\"Check if a Public IP is attached to a Port\",\"name\":\"get\",\"parametersSchema\":{\"$ref\":\"blueprint#/components/schemas/PublicIPAttachObject\"},\"resultSchema\":{\"$ref\":\"blueprint#/components/schemas/PublicIPAttachObject\"},\"scopes\":[\"network.read\"],\"steps\":[{\"check\":{\"errorMessageTemplate\":\"PublicIP {{ .parameters.public_ip_id }} is not attached to Port {{ .parameters.port_id }}\\n\",\"jsonPathQuery\":\"hasKey($.current.result, \\\"port_id\\\") && $.current.result.port_id == $.current.parameters.port_id\\n\"},\"parameters\":{\"public_ip_id\":\"$.parameters.public_ip_id\"},\"retryUntil\":{\"interval\":\"5s\",\"jsonPathQuery\":\"hasKey($.current.result, \\\"port_id\\\") && $.current.result.port_id == $.current.parameters.port_id\\n\",\"maxRetries\":3},\"target\":\"/network/public_ips/get\"}]},{\"configsSchema\":{\"$ref\":\"blueprint#/components/configsSchemas/default\"},\"confirm\":\"Detaching PublicIP {{ .parameters.public_ip_id }} from Port {{ .parameters.port_id }}. Confirm?\\n\",\"description\":\"Detach a Public IP from a Port\",\"name\":\"delete\",\"parametersSchema\":{\"$ref\":\"blueprint#/components/schemas/PublicIPAttachObject\"},\"result\":\"{\\n  \\\"public_ip_id\\\": $.parameters.public_ip_id,\\n  \\\"port_id\\\": $.parameters.port_id\\n}\\n\",\"resultSchema\":{\"$ref\":\"blueprint#/components/schemas/PublicIPAttachObject\"},\"scopes\":[\"network.write\"],\"steps\":[{\"parameters\":{\"port_id\":\"$.parameters.port_id\",\"public_ip_id\":\"$.parameters.public_ip_id\"},\"target\":\"/network/public_ips/detach\"}]}],\"description\":\"Manage the attachment between a Public IP and a Port\",\"isInternal\":true,\"name\":\"port-attachment\"}],\"description\":\"VPC Public IPs\",\"name\":\"public_ips\"},{\"children\":[{\"$ref\":\"http://magalu.cloud/sdk#/network/security_groups/rules/create\",\"isInternal\":false},{\"$ref\":\"http://magalu.cloud/sdk#/network/security_groups/rules/list\",\"isInternal\":false}],\"description\":\"VPC Rules\",\"name\":\"rules\"},{\"children\":[{\"$ref\":\"http://magalu.cloud/sdk#/network/vpcs/subnets/create\",\"isInternal\":false},{\"$ref\":\"http://magalu.cloud/sdk#/network/vpcs/subnets/list\",\"isInternal\":false}],\"description\":\"VPC Subnets\",\"name\":\"subnets\"}],\"components\":{\"configsSchemas\":{\"default\":{\"$ref\":\"/network/public_ips/get/configsSchema\"}},\"schemas\":{\"PortAttachObject\":{\"properties\":{\"port_id\":{\"$ref\":\"/network/ports/attach/parametersSchema/properties/port_id\"},\"security_group_id\":{\"$ref\":\"/network/ports/attach/parametersSchema/properties/security_group_id\"}},\"required\":[\"security_group_id\",\"port_id\"],\"type\":\"object\"},\"PublicIPAttachObject\":{\"properties\":{\"port_id\":{\"$ref\":\"/network/public_ips/attach/parametersSchema/properties/port_id\"},\"public_ip_id\":{\"$ref\":\"/network/public_ips/attach/parametersSchema/properties/public_ip_id\"}},\"required\":[\"public_ip_id\",\"port_id\"],\"type\":\"object\"}}},\"description\":\"Operations for Network API\",\"name\":\"network\",\"url\":\"https://network.magalu.cloud\",\"version\":\"1.99.5\"}"),
}

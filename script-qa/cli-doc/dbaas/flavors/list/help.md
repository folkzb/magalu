# Returns a list of available flavors. A flavor is a hardware template that defines the size of RAM and vcpu.

## Usage:
```bash
Usage:
  ./mgc dbaas flavors list [flags]
```

## Product catalog:
- Flags:
- --control.limit integer     Limit (range: 1 - 50) (default 10)
- --control.offset integer   Offset (min: 0)
- -h, --help                     help for list
- --status enum              An enumeration. (one of "ACTIVE" or "DEPRECATED") (default "ACTIVE")
- -v, --version                  version for list

## Other commands:
- Global Flags:
- --cli.show-cli-globals   Show all CLI global flags on usage text
- --env enum               Environment to use (one of "pre-prod" or "prod") (default "prod")
- --region enum            Region to reach the service (one of "br-mgl1", "br-ne1" or "br-se1") (default "br-ne1")
- --server-url uri         Manually specify the server to use

## Flags:
```bash

```

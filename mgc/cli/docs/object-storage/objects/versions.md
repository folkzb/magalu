---
sidebar_position: 17
---
# Versions

Retrieve all versions of an object

## Usage:
```
mgc object-storage objects versions [dst] [flags]
```

## Examples:
```
mgc object-storage objects versions --dst="bucket1/file.txt"
```

## Flags:
```
    --dst uri   Path of the object to retrieve versions from (required)
-h, --help      help for versions
```

## Global Flags:
```
    --api-key string           Use your API key to authenticate with the API
    --chunk-size integer       Chunk size to consider when doing multipart requests. Specified in Mb (range: 8 - 5120) (required) (default 8)
-U, --cli.retry-until string   Retry the action with the same parameters until the given condition is met. The flag parameters
                               use the format: 'retries,interval,condition', where 'retries' is a positive integer, 'interval' is
                               a duration (ex: 2s) and 'condition' is a 'engine=value' pair such as "jsonpath=expression"
-t, --cli.timeout duration     If > 0, it's the timeout for the action execution. It's specified as numbers and unit suffix.
                               Valid unit suffixes: ns, us, ms, s, m and h. Examples: 300ms, 1m30s
    --debug                    Display detailed log information at the debug level
    --no-confirm               Bypasses confirmation step for commands that ask a confirmation from the user
-o, --output string            Change the output format. Use '--output=help' to know more details.
-r, --raw                      Output raw data, without any formatting or coloring
    --region string            Region to reach the service (default "br-se1")
    --retries integer          Maximum number of retries for transient network errors (min: 0)
    --server-url uri           Manually specify the server to use
    --workers integer          Number of routines that spawn to do parallel operations within object_storage (min: 1) (required) (default 5)
```


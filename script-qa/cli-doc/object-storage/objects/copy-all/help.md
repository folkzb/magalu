# Copy all objects from a bucket to another bucket

## Usage:
```bash
Usage:
  ./mgc object-storage objects copy-all [src] [dst] [flags]
```

## Product catalog:
- Examples:
- ./mgc object-storage objects copy-all --dst="bucket2/dir/" --src="bucket1"

## Other commands:
- Flags:
- --dst uri                Full destination path in the bucket (required)
- --filter array(object)   File name pattern to include or exclude
- Use --filter=help for more details
- -h, --help                   help for copy-all
- --src uri                Path of objects in a bucket to be copied (required)

## Flags:
```bash
Global Flags:
      --chunk-size integer     Chunk size to consider when doing multipart requests. Specified in Mb (range: 8 - 5120) (default 8)
      --cli.show-cli-globals   Show all CLI global flags on usage text
      --region enum            Region to reach the service (one of "br-mgl1", "br-ne1" or "br-se1") (default "br-ne1")
      --server-url uri         Manually specify the server to use
      --workers integer        Number of routines that spawn to do parallel operations within object_storage (min: 1) (default 5)
```

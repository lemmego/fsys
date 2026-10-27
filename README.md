# lemmego/fsys

One file-storage interface over local disk, S3, Google Cloud Storage and
memory. Write your upload handling once, and change where the bytes land by
changing configuration.

```go
import "github.com/lemmego/fsys"

var disk fsys.FS = fsys.NewLocalStorage("./storage")

if err := disk.Write("uploads/report.pdf", data); err != nil {
    return err
}
url, err := disk.GetUrl("uploads/report.pdf")
```

In a Lemmego application you rarely construct one of these yourself.
`api/providers/fs` builds them from your configured disks, and
`c.App().FileSystem().Disk(name)` hands you an `app.Disk` — the same eleven
methods, declared in `api/app` so the core does not link the cloud SDKs.
Reach for the constructors directly in a test, a CLI command, or a program
that is not a Lemmego application.

## Install

```bash
go get github.com/lemmego/fsys
```

## The drivers

```go
fsys.NewMemoryStorage()
fsys.NewLocalStorage(basePath)
fsys.NewS3Storage(bucket, region, accessKey, secretKey, baseEndpoint)
fsys.NewGCSStorage(projectID, bucket, serviceAccountKey)
```

`NewS3Storage`'s `baseEndpoint` is what makes it work against anything
S3-compatible — Cloudflare R2, MinIO, Backblaze B2, DigitalOcean Spaces. Leave
it empty for AWS itself.

`NewMemoryStorage` is the one to use in tests: no temporary directory to
clean up, no ordering between tests, and `Driver()` still reports something
your code can branch on.

## The interface

Every driver satisfies `fsys.FS`:

| Method | Notes |
|---|---|
| `Write(path, contents []byte) error` | creates or replaces |
| `Read(path) (io.ReadCloser, error)` | **close it** |
| `Exists(path) (bool, error)` | |
| `Delete(path) error` | |
| `Rename(oldPath, newPath) error` | |
| `Copy(src, dst) error` | |
| `CreateDirectory(path) error` | on S3 and GCS, writes a zero-byte object with a trailing slash |
| `GetUrl(path) (string, error)` | a URL a browser can fetch |
| `Open(path) (*os.File, error)` | |
| `Upload(file multipart.File, header *multipart.FileHeader, dir string) (*os.File, error)` | straight from a submitted form |
| `Driver() string` | `memory`, `local`, `gcs` or `s3` |

Because `Write` takes a `[]byte`, a large file is held in memory. There is no
streaming write and no presigned upload URL, so a very large or
direct-to-bucket upload wants something else.

## Two things worth knowing before you serve an upload

**`Upload` writes the submitted bytes verbatim.** It does not resize, re-encode
or strip metadata — so a photograph taken on a phone keeps its EXIF block, GPS
coordinates included, and anybody who can fetch the file can read them. Put
[`lemmego/media`](https://github.com/lemmego/media) in front of an image
upload.

**Never derive a stored filename from what the client sent.** A submitted
filename is attacker-controlled: it can carry path separators, a second
extension, or a name your own code will later treat as code. Sniff the content
type and name the file from the format you detected.

## Writing your own driver

Implement `fsys.FS`. Eleven methods is more than a minimal blob-store
contract, because the interface is shaped to what an application does with
uploads rather than to what a blob store is — so a driver carries the awkward
parts instead of every caller carrying them.

`CreateDirectory` is the method to read carefully. An object store has no
directories, so the S3 and GCS drivers write a zero-byte object with a
trailing slash — which is the convention a console will display as a folder,
and is also a real object your listing and deletion code has to expect.

## Testing

```bash
go test ./...
GOWORK=off go test ./...   # as a published module
```

## License

[MIT](https://choosealicense.com/licenses/mit/)

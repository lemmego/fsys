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

Local and memory come with the root package, because neither costs you
anything — both are the standard library:

```go
fsys.NewMemoryStorage()
fsys.NewLocalStorage(basePath)
```

A cloud driver lives in its own package and registers itself when you import
it, the way a `database/sql` driver does:

```go
import _ "github.com/lemmego/fsys/s3"   // or .../gcs

disk, err := fsys.Open("s3", fsys.Settings{
    "bucket":   "uploads",
    "region":   "auto",
    "key":      os.Getenv("S3_ACCESS_KEY_ID"),
    "secret":   os.Getenv("S3_SECRET_ACCESS_KEY"),
    "endpoint": os.Getenv("S3_ENDPOINT"),
})
```

Or construct one directly, if you are not selecting it by name:

```go
store, err := s3.New(bucket, region, accessKey, secretKey, baseEndpoint)
store, err := gcs.New(projectID, bucket, serviceAccountKey)
```

**Why the split.** This package is imported by `api/fs`,
`api/providers/fs` and `media/provider`, and through them by every
application on the framework. While S3 and GCS lived in the root, all of them
linked both cloud SDKs whether or not they had ever seen a bucket. Measured on
a trivial program, importing this package was the difference between 58
packages and 691, and between a 2.3MB binary and 44MB. In one real
application it was half the binary and several seconds of every rebuild.

So you link the storage you blank-import and nothing else, and
`deps_test.go` measures that rather than trusting it.

`s3.New`'s `baseEndpoint` is what makes it work against anything
S3-compatible — Cloudflare R2, MinIO, Backblaze B2, DigitalOcean Spaces. Leave
it empty for AWS itself.

`NewMemoryStorage` is the one to use in tests: no temporary directory to
clean up, no ordering between tests, and `Driver()` still reports something
your code can branch on.

> **If you select a driver by name, import it.** A disk configured for `s3`
> without `_ "github.com/lemmego/fsys/s3"` fails when the disk is first opened,
> not at compile time — and it will pass on a development machine using the
> local disk and fail in production. The error names the drivers that *are*
> registered and tells you which import is missing, but the place to catch it
> is the blank import.

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

Implement `fsys.FS` and register it from your package's `init` with
[fsys.Register], so an application reaches it by name the way it reaches the
built-in drivers. Eleven methods is more than a minimal blob-store
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

> ⚠️ This project is still experimental. Do not use this in production.

# Fsys

Fsys is a Go library for dealing with file systems.

## Installation

Use the Go package manager to install foobar.

```bash
go get -u github.com/lemmego/fsys
```

## Usage

```go
fsys.NewMemoryStorage(...)
fsys.NewLocalStorage(...)
fsys.NewGCSStorage(...)
fsys.NewS3Storage(...)
```

## Contributing

Pull requests are welcome. For major changes, please open an issue first
to discuss what you would like to change.

Please make sure to update tests as appropriate.

## License

[MIT](https://choosealicense.com/licenses/mit/)

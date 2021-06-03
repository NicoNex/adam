# adam
Adam's Data Access Manager

## Installation
In order to install Adam you need to have the [go compiler](https://golang.org/) installed.
Once downloaded simply running this command will install or update Adam in your `$GOPATH`:
```bash
go get -u github.com/NicoNex/adam
```

To install it from sources you can clone this repo locally, run `go build` inside Adam's root directory and then place the binary in your most preferred destination.

## Usage
Once started Adam will look for the configuration files placed in `$HOME/.config/adam.toml` for Posix systems and `%UserProfile%\.config\adam.toml` for Windows systems.

The configuration file must be in the [TOML](https://toml.io/en/) format as in the following example:
```toml
base_dir = "~/custom/path/to/adam/base/directory"
port = ":8080"
cache_dir = "/home/user/.cache/adam"
```

Additionally to the configuration file Adam supports also argument flags, so if you want to specify other port/base_dir values you can run it like following:
```bash
adam -d path/to/base/dir -p 8081 -c cache
```
Run `adam --help` for additional details.


## Endpoints
All endpoints support the GET HTTP method except for the `/put` one that needs the request to be POST.

### /
This endpoint lets you browse the directory tree adam is exposing.

### /put
This endpoint lets you upload one or multiple files to a path specified in the URL.
Eg: 
```bash
curl -F 'files[]=@file1.png' -F 'files[]=@file2.webm' 'http://localhost:8080/put/example/directory' 
```

In this example Adam will place the two files `file1.png` and `file2.webm` in the path provided after `/put`.
If the directory doesn't exist Adam will create it first.

Adam will respond with a json formed like this if successful:
```json
{
	"ok": true,
	"files": [
		{
			"path": "example/directory/file1.png",
			"sha256sum": "d6663168db0e746cffeeaa8fcdc1c0486193e5e571524c202c546c743e0df7f9"
		},
		{
			"path": "example/directory/file2.png",
			"sha256sum": "5b708b81a6a9458c58d8115f0abd0ef4d692b380411c8b57c153fda9863d4184"
		}		
	]
}
```

If something went wrong with some of the files uploaded, the "ok" field will be set to `false` and the "files" array will contain only the files that have been successfully saved.

In case of errors not related to file savings (such as errors related to the checksum) the "ok" field will still be set to `true`, but the errors will be shown in the "errors" field.
This means that the the files were saved without any issues but that something has happened to some other related process. 

### /move
This endpoint lets you move (and thus also rename) a file or directory from `source` to `dest`.
Adam for this endpoint expects two query parameters called `source` and `dest`.
- `source` is the original path of the file or directory to move or rename.
- `dest` is the destination path of the file or directory to move or rename.

The response for this endpoint is the same as the response from the `/del` endpoint.

#### Move example:
In this example we move the file `file2.png` from `example/directory/file2.png` to `example/file2.png`.
```bash
curl 'http://localhost:8080/move?source=example/directory/file2.png&dest=example/file2.png'
```

#### Rename example:
In this example we rename the file `example/directory/file2.png` to `example/directory/picture2.png`.
```bash
curl 'http://localhost:8080/move?source=example/directory/file2.png&dest=example/directory/picture2.png'
```

### /del
This endpoint lets you delete a file or a directory.
If successful you'll be presented with a response formed like this:
```json
{
	"ok": true
}
```

Otherwise if not successful Adam will include the error description in the response.
Eg:
```json
{
	"ok": false,
	"error": "file not found"
}
```

### /sha256sum
This endpoint returns the sha256sum of the file at the path specified after the endpoint name.
Eg:
```bash
curl 'http://localhost:8080/sha256sum/example/adam'
```
Will result in:
```json
{
	"ok": true,
	"file": "example/adam",
	"sha256sum": "d6663168db0e746cffeeaa8fcdc1c0486193e5e571524c202c546c743e0df7f9"
}

```

In case an error happens, the response will be as for the /del endpoint.
Eg:
```bash
curl 'http://localhost:8080/sha256sum/example/adam/testError'
```
Will result in:
```json
{
	"ok": false,
	"error": "no sha256sum for path example/adam/testError"
}
```

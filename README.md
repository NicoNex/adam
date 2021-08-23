# adam
Adam's Data Access Manager

[![Go Report Card](https://goreportcard.com/badge/github.com/NicoNex/adam)](https://goreportcard.com/report/github.com/NicoNex/adam)

## Installation
In order to install Adam you need to have the [go compiler](https://golang.org/) installed.
Once downloaded simply running this command will install or update Adam in your `$GOPATH`:
```bash
$ go get -u github.com/NicoNex/adam
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
$ adam -d path/to/base/dir -p 8081 -c cache
```
Run `adam --help` for additional details.


## Endpoints
All endpoints support the GET HTTP method except for the `/put` one that needs the request to be POST.

### /
This endpoint lets you browse the directory tree adam is exposing.

### /get
This endpoint lets you download a file given its ID.

Eg:
```bash
curl 'http://localhost:8080/get?id=959aec06-edfb-4efa-a114-2fbb8ee9dd29'
```
This way Adam will serve you the file if found or an error similar to the /del method if something went wrong.

### /put
This endpoint lets you upload one or multiple files to a path specified in the URL.

Eg: 
```bash
$ curl -F 'files[]=@file1.png' -F 'files[]=@file2.webm' 'http://localhost:8080/put/example/directory' 
```

In this example Adam will place the two files `file1.png` and `file2.webm` in the path provided after `/put`.
If the directory doesn't exist Adam will create it first.

Adam will respond with a json formed like this if successful:
```json
{
  "ok": true,
  "files": [
    {
      "path":"example/directory/file1.png",
      "sha256sum":"0c15e883dee85bb2f3540a47ec58f617a2547117f9096417ba5422268029f501",
      "id":"959aec06-edfb-4efa-a114-2fbb8ee9dd29"
    },
    {
      "path":"example/directory/file2.webm",
      "sha256sum":"19cf8915f014fec66ebef02e6bd0de82e4591514165ea68a95b2ad71ac119fb2",
      "id":"077b7b79-1262-45ba-a13a-cac61df3ff06"
    },
  ]
}
```

If something went wrong with some of the files uploaded, the "ok" field will be set to `false` and the "files" array will contain only the files that have been successfully saved.
In such case the response json will additionally have an `errors` field containing all the errors encountered while saving the files like in the following example:

Eg: 
```bash
$ curl \
	-F 'files[]=@file1.png' \
	-F 'files[]=@file2.webm' \
	-F 'files[]=@file3.mp4' \
	'http://localhost:8080/put/example/directory' 
```
```json
{
  "ok": false,
  "files": [
    {
      "path":"example/directory/file1.png",
      "sha256sum":"0c15e883dee85bb2f3540a47ec58f617a2547117f9096417ba5422268029f501",
      "id":"959aec06-edfb-4efa-a114-2fbb8ee9dd29"
    },
    {
      "path":"example/directory/file2.webm",
      "sha256sum":"19cf8915f014fec66ebef02e6bd0de82e4591514165ea68a95b2ad71ac119fb2",
      "id":"077b7b79-1262-45ba-a13a-cac61df3ff06"
    },
  ],
  "errors": [
  	"could not save file example/directory/file3.mp4"
  ],
}
```

### /move
This endpoint lets you move (and thus also rename) a file or directory from `oldpath` to `newpath`.
Adam for this endpoint expects two query parameters called `oldpath` and `newpath`.
- `oldpath` is the original path of the file or directory to move or rename.
- `newpath` is the destination path of the file or directory to move or rename.

Optionally instead of `oldpath` you can provide Adam the `id` query parameter with the right file ID. 
The response for this endpoint is the same as the response from the `/del` endpoint.

#### Move example:
In this example we move the file `file2.png` from `example/directory/file2.png` to `example/file2.png`.
```bash
$ curl 'http://localhost:8080/move?oldpath=example/directory/file2.png&newpath=example/file2.png'
```

#### Rename example:
In this example we rename the file `example/directory/file2.png` to `example/directory/picture2.png`.
```bash
$ curl 'http://localhost:8080/move?oldpath=example/directory/file2.png&newpath=example/directory/picture2.png'
```

#### Move with ID example:
In this example we will move the file referenced by the ID `077b7b79-1262-45ba-a13a-cac61df3ff06` to `example/directory/picture2.png`.
```bash
$ curl 'http://localhost:8080/move?id=077b7b79-1262-45ba-a13a-cac61df3ff06&newpath=example/file2.png'
```

### /del
This endpoint lets you delete a file or a directory.

Eg:
```bash
$ curl 'https://localhost:8080/del/example/directory/picture2.png'
```

Optionally instead of specifying the file path you can provide its ID.

Eg:
```bash
$ curl 'https://localhost:8080/del?id=077b7b79-1262-45ba-a13a-cac61df3ff06'
```

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
$ curl 'http://localhost:8080/sha256sum/example/adam'
```

Optionally instead of specifying the file path you can provide its ID.

Eg:
```bash
$ curl 'https://localhost:8080/sha256sum?id=077b7b79-1262-45ba-a13a-cac61df3ff06'
```

If succesful the response will be formed like in the following example:
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
$ curl 'http://localhost:8080/sha256sum/example/adam/testError'
```
Will result in:
```json
{
  "ok": false,
  "error": "no sha256sum for path example/adam/testError"
}
```

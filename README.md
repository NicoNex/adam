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
```

Additionally to the configuration file Adam supports also argument flags, so if you want to specify other port/base_dir values you can run it like following:
```bash
adam -d path/to/base/dir -p 8081
```
Run `adam --help` for additional details.


## Endpoints

### /
This endpoint lets you browse the directory tree adam is exposing.

### /put
This endpoint let's you upload one or multiple files to a path specified in the URL.
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
		"example/directory/file1.png",
		"example/directory/file2.png"
	]
}
```

If something went wrong with some of the files uploaded, the "ok" field will be set to `false` and the "files" array will contain only the files that have been successfully saved.


### /del
This endpoint let's you delete a file or a directory.
If successful you'll be presented with a response formed like this:
```json
{
	"ok": true
}
```

Otherwise if not successful Adam will include the error description in the response:
```json
{
	"ok": false,
	"error": "file not found"
}
```

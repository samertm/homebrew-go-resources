homebrew-go-resources generates `go_resource` statements for homebrew
formulas. It generates `go_resource` statements for the currently
checked out repos for your project and prints them on stdout.
It works for 'hg' and 'git' repositories.

Before running this tool, you should fetch the projet by either
running `go get -u project/import/path` (to fetch the latest version)
or, to fetch a specific release:

* `git clone https://url/to/project src/project/import/path`
* `cd src/project/import/path`
* `git checkout tag # for example, v1.0.0`
* `go get -d`


```
Usage of homebrew-go-resources:
	homebrew-go-resources [flags] [path]
Flags:
  -debug=false: show debug messages
```


### Install

```
go get github.com/samertm/homebrew-go-resources
```

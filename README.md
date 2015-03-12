homebrew-go-resources generates `go_resource` statements for homebrew
formulas. It generates `go_resource` statements for the currently
checked out repos for your project and prints them on stdout. You
should run `go get -u project/import/path` before running
homebrew-go-resources. It works for 'hg' and 'git' repositories.

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

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/build"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

var debugOpt bool

func init() {
	flag.BoolVar(&debugOpt, "debug", false, "show debug messages")
}

// The dLogger is used for debug logs.
var dLogger = log.New(os.Stderr, "", 0)

func dLogf(format string, v ...interface{}) {
	if !debugOpt {
		return
	}
	dLogger.Printf(format, v...)
}

func dLog(v ...interface{}) {
	if !debugOpt {
		return
	}
	dLogger.Println(v...)
}

func usage() {
	msg := `Usage of homebrew-go-resources:
        homebrew-go-resources [flags] [path]

homebrew-go-resources generates "go_resource" statements for homebrew
formulas. It generates "go_resource" statements for the currently
checked out repos for your project. It works for 'hg' and 'git'
repositories.

Flags:`
	fmt.Fprintf(os.Stderr, msg)
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	flag.Usage = usage
	flag.Parse()
	var projectImportPath string
	args := flag.Args()
	if len(args) > 1 {
		log.Fatal("error: too many args")
	} else if len(args) == 1 {
		projectImportPath = args[0]
	}
	log.SetFlags(log.Lshortfile)
	cmd := exec.Command("go", "list", "-e", "-json")
	if projectImportPath != "" {
		cmd.Args = append(cmd.Args, projectImportPath)
	}
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	var listOut struct {
		Dir        string
		ImportPath string
		Deps       []string
	}
	if err := json.NewDecoder(stdout).Decode(&listOut); err != nil {
		log.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
	if len(listOut.Deps) == 0 {
		log.Fatal("No deps found. If this is unexpected, please file a bug report.")
	}
	type vcsInfo struct {
		ImportPath string
		ClonePath  string
		Revision   string
		VCS        string
	}
	// seen maps import paths to vcsInfos.
	seen := map[string]vcsInfo{}
	var allInfo []vcsInfo
	// Process the current project as the first 'dep', because we
	// treat it specially (it doesn't get printed out/added to
	// allInfo).
	listOut.Deps = append([]string{listOut.ImportPath}, listOut.Deps...)
	for depIndex, dep := range listOut.Deps {
		dLog("Importing dep", dep)
		pkg, err := build.Default.Import(dep, "", build.FindOnly)
		if err != nil {
			log.Fatal(err)
		}
		if pkg.Goroot {
			dLog("In Goroot, continuing...")
			continue
		}
		// Try to find Git repository first.
		cmd := exec.Command("git", "rev-parse", "--show-toplevel")
		cmd.Dir = filepath.Join(build.Default.GOPATH, "src", pkg.ImportPath)
		out, err := cmd.Output()
		if err != nil {
			// If Git fails, try Mercurial.
			cmd := exec.Command("hg", "root")
			cmd.Dir = filepath.Join(build.Default.GOPATH, "src", pkg.ImportPath)
			o, err := cmd.Output()
			if err != nil {
				log.Fatalf("Could not find 'git' or 'hg' repo for %s", dep)
			}
			out = o
		}
		dir := strings.TrimSuffix(string(out), "\n")
		dLogf("Operating on top level dir '%s'", dir)
		var i vcsInfo
		p, err := filepath.Rel(filepath.Join(build.Default.GOPATH, "src"), dir)
		if err != nil {
			log.Fatal(err)
		}
		i.ImportPath = p
		dLogf("Import path %s", p)
		if _, ok := seen[i.ImportPath]; ok {
			dLog("Seen, continuing...")
			continue
		}
		// First, we check to see if it's a git repository.
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			i.VCS = "git"
			cmd := exec.Command("git", "remote", "-v")
			cmd.Dir = dir
			out, err := cmd.Output()
			if err != nil {
				log.Fatal(err)
			}
			for _, line := range strings.Split(string(out), "\n") {
				if strings.HasPrefix(line, "origin") {
					clone := strings.Fields(line)[1]
					if index := strings.Index(clone, "@"); index != -1 {
						clone = "https://" + clone[index+1:]
						clone = strings.Replace(clone, ":", "/", 1)
					}
					i.ClonePath = clone
				}
			}
			if i.ClonePath == "" {
				log.Fatalf("Could not find a clone path for %s. Please file a bug report", i.ImportPath)
			}
			cmd = exec.Command("git", "rev-parse", "HEAD")
			cmd.Dir = dir
			out, err = cmd.Output()
			if err != nil {
				log.Fatal(err)
			}
			i.Revision = strings.TrimSuffix(string(out), "\n")
		}
		if _, err := os.Stat(filepath.Join(dir, ".hg")); err == nil {
			i.VCS = "hg"
			cmd := exec.Command("hg", "paths", "default")
			cmd.Dir = dir
			out, err := cmd.Output()
			if err != nil {
				log.Fatal(err)
			}
			clone := strings.TrimSuffix(string(out), "\n")
			if index := strings.Index(clone, "@"); index != -1 {
				clone = "https://" + clone[index+1:]
				clone = strings.Replace(clone, ":", "/", 1)
			}
			i.ClonePath = clone
			if i.ClonePath == "" {
				log.Fatalf("Could not find a clone path for %s. Please file a bug report", i.ImportPath)
			}
			cmd = exec.Command("hg", "identify", "--debug", "-i")
			cmd.Dir = dir
			out, err = cmd.Output()
			if err != nil {
				log.Fatal(err)
			}
			i.Revision = strings.TrimSuffix(string(out), "\n")
		}
		if i.VCS == "" {
			log.Fatalf("Could not find vcs for %s. If this is unexpected, please file a bug report. %+v", i.ImportPath, i)
		}
		// As a special case, add ".git" to the end of git remotes.
		// Fixes https://github.com/samertm/homebrew-go-resources/issues/1.
		if i.VCS == "git" && !strings.HasSuffix(i.ClonePath, ".git") {
			i.ClonePath += ".git"
		}
		if strings.HasSuffix(i.ClonePath, "."+i.VCS) {
			i.VCS = ""
		}
		seen[i.ImportPath] = i
		// Do not add the current project to allInfo (it is
		// prepended to Deps).
		if depIndex != 0 {
			allInfo = append(allInfo, i)
		}
		// Get the toplevel dir and cut the GOPATH off the
		// front to get the import path.
	}
	if err := templateOut.Execute(os.Stdout, allInfo); err != nil {
		log.Fatal(err)
	}
}

var templateString = `
{{range .}}
  go_resource "{{.ImportPath}}" do
    url "{{.ClonePath}}",
        :revision => "{{.Revision}}"{{if .VCS}}, :using => :{{.VCS}}{{end}}
  end
{{end}}
`

var templateOut = template.Must(template.New("out").Parse(templateString))

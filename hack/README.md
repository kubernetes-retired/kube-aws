## Release note gathering

To generate a release note for the release `v0.9.8-rc.1` in Markdown:

```
$ go get -u "github.com/google/go-github/github"
$ go get -u "golang.org/x/oauth2"
$ go get -u "golang.org/x/context"
$ VERSION=v0.9.8-rc.1 go run relnotes.go
```

# watch

Watch .go and .mod files, run `go build` or `go test` command if these files changed.

```go
import "github.com/gopsql/logger"
import "github.com/gopsql/watch"

logger := logger.StandardLogger
logger.Fatal(watch.NewWatch().WithLogger(logger).Do())
```

## gow command

Install:

```
go install -v github.com/gopsql/watch/gow@latest
```

Run:

```
# this watches all go files in current directory:
gow

# gow by default ignores node_modules, .git, dist,
# to ignore extra directory names:
gow -ignore vendor -ignore another-dir

# to add extra go build arguments:
gow -- -v -race -o another-name

# to add extra app run arguments:
gow -- -v -race -o another-name -- --my-app-customer-flag

# clean test cache before running "go test -v ./..." in "tests" directory
gow -cd tests -test -clean -- -v ./...
```

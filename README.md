# watch

Watch .go and .mod files, run `go build` or `go test` command if these files changed.

```go
import "github.com/gopsql/logger"
import "github.com/gopsql/watch"

logger := logger.StandardLogger
logger.Fatal(watch.NewWatch().WithLogger(logger).Do())
```

## gow command

If you want a command line program, see <https://github.com/gopsql/gow>.

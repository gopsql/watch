# watch

Watch .go and .mod files, run `go build` command and the newly built executable
if these files changed.

```go
import "github.com/gopsql/logger"
import "github.com/gopsql/watch"

logger := logger.StandardLogger
logger.Fatal(watch.NewWatch().WithLogger(logger).Do())
```

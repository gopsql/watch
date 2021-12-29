# watch

```
import "github.com/gopsql/logger"
import "github.com/gopsql/watch"
logger := logger.StandardLogger
logger.Fatal(watch.NewWatch().WithLogger(logger).Do())
```

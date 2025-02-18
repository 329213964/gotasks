# gotasks

[![Build Status](https://travis-ci.org/jiajunhuang/gotasks.svg?branch=master)](https://travis-ci.org/jiajunhuang/gotasks)
[![codecov](https://codecov.io/gh/jiajunhuang/gotasks/branch/master/graph/badge.svg)](https://codecov.io/gh/jiajunhuang/gotasks)

gotasks is a task/job queue framework for Golang. Currently we support use Redis as broker, but you can replace it
easily by implement `Broker` interface in `broker.go`.

In gotasks, we encourage developer to split tasks into smaller pieces(see the demo bellow) so we can:

- maintain tasks easily
- split code into reentrant and un-reentrant pieces, so when reentrant part failed, framework will retry it automatically
- concurrency control

As handlers are chained, ArgsMap we give will be arguments to the first handler, and ArgsMap in it's return value will
be arguments to the second handler. If any handler return a error, the execution chain will stop, and record where it
is now, with the error string inside the task's `result_log` property.

# Usage

- producer generate tasks to broker:

```go
package main

import (
	"github.com/329213964/gotasks"
	//"github.com/329213964/gotasks/metrics"
)

const (
	uniqueJobName = "a-unique-job-name"
	redisURL      = "redis://127.0.0.1:6379/0"
	queueName     = "job-queue-name"
)

func main() {
	// set broker
	gotasks.UseRedisBroker(redisURL, gotasks.WithRedisTaskTTL(1000))

	// enqueue
	// or you can use a queue:
	queue := gotasks.NewQueue(queueName, gotasks.WithMaxLimit(10))
	queue.Enqueue(uniqueJobName, gotasks.MapToArgsMap(map[string]interface{}{})) // or gotasks.StructToArgsMap
}
```

- consumer consumes tasks from broker:

```go
package main

import (
	"context"
	"time"

	"github.com/329213964/gotasks"
	//"github.com/329213964/gotasks/metrics"
)

const (
	uniqueJobName = "a-unique-job-name"
	redisURL      = "redis://127.0.0.1:6379/0"
	queueName     = "job-queue-name"
)

func Handler1(args gotasks.ArgsMap) (gotasks.ArgsMap, error) {
	time.Sleep(time.Duration(1) * time.Second)
	return args, nil
}
func Handler2(args gotasks.ArgsMap) (gotasks.ArgsMap, error) {
	time.Sleep(time.Duration(1) * time.Second)
	return args, nil
}

func register() {
	// if handler1 failed, the task will stop, but if handler2 failed(return a non-nil error)
	// handler2 will be retry 3 times, and sleep 100 ms each time
	gotasks.Register(uniqueJobName, Handler1, gotasks.Reentrant(Handler2, gotasks.WithMaxTimes(3), gotasks.WithSleepyMS(10)))
}

func worker() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gotasks.Run(ctx, queueName)
}

func main() {
	// set broker
	gotasks.UseRedisBroker(redisURL, gotasks.WithRedisTaskTTL(1000))

	register()
	worker()
}
```

## License

NOTE that from the first commit, we use a GPL-v3 open source license.

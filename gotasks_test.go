package gotasks

import (
	"context"
	"errors"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	testJobName                = "test_job"
	testPanicJobName           = "test_panic_job"
	testArgsPassJobName        = "test_args_pass_job"
	testReentrantJobName       = "test_reentrant_job"
	testHandlerNotFoundJobName = "test_handler_not_found"

	testQueueName = "test_queue"
	testRedisURL  = "redis://:QAXRedisServer@127.0.0.1:6379/0"
)

func TestAckWhen(t *testing.T) {
	AckWhen(AckWhenAcquired)
	AckWhen(AckWhenSucceed)
}

func TestGenFunctions(t *testing.T) {
	assert.Equal(t, "gt:task:abcd", genTaskName("abcd"))
	assert.Equal(t, "gt:queue:abcd", genQueueName("abcd"))
}

func TestRedisBroker(t *testing.T) {
	//// register tasks
	//handler1 := func(args ArgsMap) (ArgsMap, error) {
	//	time.Sleep(time.Duration(1) * time.Microsecond)
	//	return args, nil
	//}
	//handler2 := func(args ArgsMap) (ArgsMap, error) {
	//	time.Sleep(time.Duration(1) * time.Microsecond)
	//	return args, nil
	//}
	//Register(testJobName, handler1, handler2)
	//
	// set broker
	UseRedisBroker(testRedisURL, WithRedisTaskTTL(100))

	// enqueue
	log.Printf("current jobMap: %+v", jobMap)
	queue := NewQueue(testQueueName, WithMaxLimit(20), WithMonitorInterval(5))
	queue.Enqueue(testJobName, MapToArgsMap(map[string]interface{}{}))
	//defer rc.Del(genTaskName(taskID))

	ctx, cancel := context.WithCancel(context.Background())
	//go Run(ctx, testQueueName) // it will blocking until the first job is executed
	time.Sleep(time.Second * time.Duration(1))
	cancel()
	log.Printf("Run function returned, ctx: %+v", ctx)
}

func TestQueue_Enqueue(t *testing.T) {
	UseRedisBroker(testRedisURL, WithRedisTaskTTL(100))

	// enqueue
	log.Printf("current jobMap: %+v", jobMap)
	queue := NewQueue(testQueueName, WithMaxLimit(20), WithMonitorInterval(5))
	queue.Enqueue(testJobName, MapToArgsMap(map[string]interface{}{}))

	ctx, cancel := context.WithCancel(context.Background())
	time.Sleep(time.Second * time.Duration(1))
	cancel()
	log.Printf("Run function returned, ctx: %+v", ctx)
}

func TestRedisBroker_Acquire(t *testing.T) {
	// register tasks
	handler1 := func(args ArgsMap) (ArgsMap, error) {
		time.Sleep(time.Duration(1) * time.Microsecond)
		fmt.Println(args)
		return args, nil
	}
	handler2 := func(args ArgsMap) (ArgsMap, error) {
		time.Sleep(time.Duration(1) * time.Microsecond)
		fmt.Println(args)
		return args, nil
	}
	Register(testJobName, handler1, handler2)

	// set broker
	UseRedisBroker(testRedisURL, WithRedisTaskTTL(100))
	ctx, cancel := context.WithCancel(context.Background())
	go Run(ctx, testQueueName) // it will blocking until the first job is executed
	time.Sleep(time.Second * time.Duration(1))
	cancel()
}

func TestPanicHandler(t *testing.T) {
	// register tasks
	handler1 := func(args ArgsMap) (ArgsMap, error) {
		time.Sleep(time.Duration(1) * time.Microsecond)
		return args, nil
	}
	handler2 := func(args ArgsMap) (ArgsMap, error) {
		time.Sleep(time.Duration(1) * time.Microsecond)
		panic("whoops")
		//return args, nil
	}
	handler3 := func(args ArgsMap) (ArgsMap, error) {
		time.Sleep(time.Duration(1) * time.Microsecond)
		return args, nil
	}
	Register(testPanicJobName, handler1, handler2, handler3)

	// set broker
	UseRedisBroker(testRedisURL)

	// enqueue
	log.Printf("current jobMap: %+v", jobMap)
	queue := NewQueue(testQueueName)
	taskID := queue.Enqueue(testPanicJobName, MapToArgsMap(map[string]interface{}{}))
	defer rc.Del(genTaskName(taskID))

	ctx, cancel := context.WithCancel(context.Background())
	go Run(ctx, testQueueName) // it will blocking until the first job is executed
	time.Sleep(time.Second * time.Duration(1))
	cancel()
	log.Printf("Run function returned, ctx: %+v", ctx)

	// check result
	taskBytes := []byte{}
	if err := rc.Get(genTaskName(taskID)).Scan(&taskBytes); err != nil {
		t.Logf("failed to get task %s: %s", taskID, err)
		t.FailNow()
	}
	task := Task{}
	if err := json.Unmarshal(taskBytes, &task); err != nil {
		t.Logf("failed to get task %s: %s", taskID, err)
		t.FailNow()
	}
	assert.Equal(t, 1, task.CurrentHandlerIndex)

	// check result ttl
	duration, err := rc.TTL(genTaskName(taskID)).Result()
	if err != nil {
		t.Logf("task %s should have ttl with err %s", taskID, err)
		t.FailNow()
	}
	if duration.Seconds() == 0 {
		t.Logf("task %s should have ttl but not", taskID)
		t.FailNow()
	}
}

func TestArgsPass(t *testing.T) {
	// register tasks
	handler1 := func(args ArgsMap) (ArgsMap, error) {
		time.Sleep(time.Duration(1) * time.Microsecond)
		args["hello"] = "world"
		return args, nil
	}
	handler2 := func(args ArgsMap) (ArgsMap, error) {
		time.Sleep(time.Duration(1) * time.Microsecond)
		assert.Equal(t, "world", args["hello"])
		return args, nil
	}
	Register(testArgsPassJobName, handler1, handler2)

	// set broker
	UseRedisBroker(testRedisURL)

	// enqueue
	log.Printf("current jobMap: %+v", jobMap)
	queue := NewQueue(testQueueName)
	taskID := queue.Enqueue(testArgsPassJobName, MapToArgsMap(map[string]interface{}{}))
	defer rc.Del(genTaskName(taskID))

	ctx, cancel := context.WithCancel(context.Background())
	go Run(ctx, testQueueName) // it will blocking until the first job is executed
	time.Sleep(time.Second * time.Duration(1))
	cancel()
	log.Printf("Run function returned, ctx: %+v", ctx)
}

func TestReentrant(t *testing.T) {
	// register tasks
	handler1 := func(args ArgsMap) (ArgsMap, error) {
		time.Sleep(time.Duration(1) * time.Microsecond)
		args["hello"] = "world"
		return args, nil
	}
	handler2 := func(args ArgsMap) (ArgsMap, error) {
		return args, errors.New("hello world error")
	}

	Register(testReentrantJobName, handler1, Reentrant(handler2, WithMaxTimes(3), WithSleepyMS(10)))

	// set broker
	UseRedisBroker(testRedisURL)

	// enqueue
	log.Printf("current jobMap: %+v", jobMap)
	queue := NewQueue(testQueueName)
	taskID := queue.Enqueue(testReentrantJobName, MapToArgsMap(map[string]interface{}{}))
	defer rc.Del(genTaskName(taskID))

	ctx, cancel := context.WithCancel(context.Background())
	go Run(ctx, testQueueName) // it will blocking until the first job is executed
	time.Sleep(time.Second * time.Duration(1))
	cancel()
	log.Printf("Run function returned, ctx: %+v", ctx)
}

func TestJobHandlerNotFound(t *testing.T) {
	// set broker
	UseRedisBroker(testRedisURL)

	// enqueue
	log.Printf("current jobMap: %+v", jobMap)
	queue := NewQueue(testQueueName)
	taskID := queue.Enqueue(testHandlerNotFoundJobName, MapToArgsMap(map[string]interface{}{}))
	defer rc.Del(genTaskName(taskID))

	ctx, cancel := context.WithCancel(context.Background())
	go Run(ctx, testQueueName) // it will blocking until the first job is executed
	time.Sleep(time.Second * time.Duration(1))
	cancel()
	log.Printf("Run function returned, ctx: %+v", ctx)
}

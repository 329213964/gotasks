package gotasks

import (
	"log"
	"time"

	redis "github.com/go-redis/redis/v7"
)

var (
	_ Broker = &RedisBroker{}

	// rc: RedisClient
	rc *redis.Client
)

func genTaskName(taskID string) string {
	return "gt:task:" + taskID
}

func genQueueName(queueName string) string {
	return "gt:queue:" + queueName
}

type RedisBroker struct {
	TaskTTL int
}

type RedisBrokerOption func(rb *RedisBroker)

func WithRedisTaskTTL(ttl int) RedisBrokerOption {
	return func(rb *RedisBroker) {
		rb.TaskTTL = ttl
	}
}

func UseRedisBroker(redisURL string, brokerOptions ...RedisBrokerOption) {
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Panicf("failed to parse redis URL %s: %s", redisURL, err)
	}

	rc = redis.NewClient(options)
	rb := &RedisBroker{}
	for _, o := range brokerOptions {
		o(rb)
	}

	broker = rb
}

func (r *RedisBroker) Acquire(queueName string) *Task {
	task := Task{}
	vs, err := rc.BRPop(time.Duration(0), genQueueName(queueName)).Result()
	if err != nil {
		log.Panicf("failed to get task from redis: %s", err)
		return nil // never executed
	}
	v := []byte(vs[1])

	result, err := rc.Get(genTaskName(string(v))).Result()
	if err != nil {
		log.Panicf("failed to get task from redis: %s", err)
		return nil // never executed
	}

	if err = json.Unmarshal([]byte(result), &task); err != nil {
		log.Panicf("failed to get task from redis: %s", err)
		return nil // never executed
	}

	return &task
}

func (r *RedisBroker) Ack(task *Task) bool {
	cmd := rc.Del(genTaskName(task.ID))
	if cmd.Err() != nil {
		log.Panicf("failed to del task id is:%s", cmd.Err())
		return false
	}
	return true
}

func (r *RedisBroker) Update(task *Task) {
	task.UpdatedAt = time.Now()
	taskBytes, err := json.Marshal(task)
	if err != nil {
		log.Panicf("failed to enquue task %+v: %s", task, err)
		return // never executed here
	}
	rc.Set(genTaskName(task.ID), taskBytes, time.Duration(r.TaskTTL)*time.Second)
}

func (r *RedisBroker) Enqueue(task *Task) string {
	taskBytes, err := json.Marshal(task)
	if err != nil {
		log.Panicf("failed to enquue task %+v: %s", task, err)
		return "" // never executed here
	}

	rc.Set(genTaskName(task.ID), taskBytes, time.Duration(r.TaskTTL)*time.Second)
	rc.LPush(genQueueName(task.QueueName), task.ID)
	return task.ID
}

func (r *RedisBroker) QueueLen(queueName string) int64 {
	l, _ := rc.LLen(genQueueName(queueName)).Result()
	return l
}

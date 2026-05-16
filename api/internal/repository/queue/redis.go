package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/kilip/opus/api/internal/model"
)

type redisDriver struct {
	client *redis.Client
	prefix string
}

// NewRedisDriver creates a new Redis-based queue driver.
func NewRedisDriver(client *redis.Client, prefix string) QueueDriver {
	if prefix == "" {
		prefix = "opus"
	}
	return &redisDriver{
		client: client,
		prefix: prefix,
	}
}

// Push persists a job to the Redis queue.
func (d *redisDriver) Push(ctx context.Context, m *model.Job) error {
	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	pipe := d.client.Pipeline()
	pipe.Set(ctx, d.jobKey(m.ID), data, 0)
	pipe.ZAdd(ctx, d.pendingKey(), redis.Z{
		Score:  float64(m.ScheduledAt.UnixNano()),
		Member: m.ID,
	})

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to push job to redis: %w", err)
	}
	return nil
}

func (d *redisDriver) Pop(ctx context.Context) (*model.Job, error) {
	script := `
		local id = redis.call('ZRANGEBYSCORE', KEYS[1], '-inf', ARGV[1], 'LIMIT', 0, 1)[1]
		if id then
			redis.call('ZREM', KEYS[1], id)
			return id
		end
		return nil
	`
	now := time.Now().UnixNano()
	res, err := d.client.Eval(ctx, script, []string{d.pendingKey()}, now).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to pop job from redis: %w", err)
	}

	if res == nil {
		return nil, nil
	}

	id := res.(string)
	data, err := d.client.Get(ctx, d.jobKey(id)).Bytes()
	if err != nil {
		return nil, fmt.Errorf("failed to get job data from redis: %w", err)
	}

	var m model.Job
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	m.Status = model.StatusRunning
	return &m, nil
}

func (d *redisDriver) UpdateStatus(ctx context.Context, id string, status model.JobStatus, errMsg string) error {
	data, err := d.client.Get(ctx, d.jobKey(id)).Bytes()
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	var m model.Job
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("failed to unmarshal job: %w", err)
	}

	m.Status = status
	m.Error = errMsg
	m.UpdatedAt = time.Now()

	newData, _ := json.Marshal(m)
	return d.client.Set(ctx, d.jobKey(id), newData, 0).Err()
}

func (d *redisDriver) MoveToDead(ctx context.Context, m *model.Job) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}

	pipe := d.client.Pipeline()
	pipe.RPush(ctx, d.deadKey(), data)
	pipe.Del(ctx, d.jobKey(m.ID))
	pipe.ZRem(ctx, d.pendingKey(), m.ID)

	_, err = pipe.Exec(ctx)
	return err
}


func (d *redisDriver) UpsertCron(ctx context.Context, m *model.CronSchedule) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return d.client.HSet(ctx, d.cronKey(), m.Name, data).Err()
}

func (d *redisDriver) DeleteCron(ctx context.Context, id string) error {
	// Note: Redis driver uses Name as key for HSet in this implementation
	// We might need to adjust if ID is strictly required.
	// For now, let's assume Name is unique as per EntGo schema.
	return d.client.HDel(ctx, d.cronKey(), id).Err()
}

func (d *redisDriver) ListPendingCrons(ctx context.Context) ([]*model.CronSchedule, error) {
	all, err := d.client.HGetAll(ctx, d.cronKey()).Result()
	if err != nil {
		return nil, err
	}

	var result []*model.CronSchedule
	now := time.Now()
	for _, data := range all {
		var c model.CronSchedule
		if err := json.Unmarshal([]byte(data), &c); err != nil {
			continue
		}
		if c.IsActive && (c.NextRunAt.IsZero() || c.NextRunAt.Before(now)) {
			result = append(result, &c)
		}
	}
	return result, nil
}

func (d *redisDriver) UpdateCronNextRun(ctx context.Context, id string, lastRun, nextRun time.Time) error {
	// Again, using ID as name for HDel/HSet consistency
	data, err := d.client.HGet(ctx, d.cronKey(), id).Result()
	if err != nil {
		return err
	}

	var c model.CronSchedule
	if err := json.Unmarshal([]byte(data), &c); err != nil {
		return err
	}

	c.LastRunAt = lastRun
	c.NextRunAt = nextRun
	c.UpdatedAt = time.Now()

	newData, _ := json.Marshal(c)
	return d.client.HSet(ctx, d.cronKey(), id, newData).Err()
}

func (d *redisDriver) ListDeadLetters(ctx context.Context, limit, offset int) ([]*model.DeadLetter, error) {
	start := int64(offset)
	stop := int64(offset + limit - 1)

	items, err := d.client.LRange(ctx, d.deadKey(), start, stop).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list dead letters: %w", err)
	}

	result := make([]*model.DeadLetter, len(items))
	for i, data := range items {
		var m model.Job
		if err := json.Unmarshal([]byte(data), &m); err != nil {
			continue
		}
		result[i] = &model.DeadLetter{
			ID:        fmt.Sprintf("dl-%s", m.ID),
			JobID:     m.ID,
			Type:      m.Type,
			Payload:   m.Payload,
			LastError: m.Error,
			Retries:   m.Retries,
			CreatedAt: m.CreatedAt,
		}
	}
	return result, nil
}

func (d *redisDriver) RetryDeadLetter(ctx context.Context, id string) error {
	// For Redis, the id passed here is actually the job ID (stripped of prefix)
	// or we need to find it in the list.
	// Simpler implementation: we find the job in dead list by ID and move it back.
	// But ListDeadLetters returns a synthetic ID "dl-ID". Let's handle both.
	jobID := id
	if len(id) > 3 && id[:3] == "dl-" {
		jobID = id[3:]
	}

	items, err := d.client.LRange(ctx, d.deadKey(), 0, -1).Result()
	if err != nil {
		return err
	}

	for _, data := range items {
		var m model.Job
		if err := json.Unmarshal([]byte(data), &m); err != nil {
			continue
		}
		if m.ID == jobID {
			// Found it. Move back to pending.
			m.Status = model.StatusPending
			m.Retries = 0
			m.ScheduledAt = time.Now()

			pipe := d.client.Pipeline()
			pipe.LRem(ctx, d.deadKey(), 1, data)
			
			newData, _ := json.Marshal(m)
			pipe.Set(ctx, d.jobKey(m.ID), newData, 0)
			pipe.ZAdd(ctx, d.pendingKey(), redis.Z{
				Score:  float64(m.ScheduledAt.UnixNano()),
				Member: m.ID,
			})

			_, err := pipe.Exec(ctx)
			return err
		}
	}

	return fmt.Errorf("dead letter not found: %s", id)
}

// DeleteDeadLetter removes a dead letter job without retrying.
func (d *redisDriver) DeleteDeadLetter(ctx context.Context, id string) error {
	jobID := id
	if len(id) > 3 && id[:3] == "dl-" {
		jobID = id[3:]
	}

	items, err := d.client.LRange(ctx, d.deadKey(), 0, -1).Result()
	if err != nil {
		return err
	}

	for _, data := range items {
		var m model.Job
		if err := json.Unmarshal([]byte(data), &m); err != nil {
			continue
		}
		if m.ID == jobID {
			return d.client.LRem(ctx, d.deadKey(), 1, data).Err()
		}
	}
	return nil
}

// Helpers
func (d *redisDriver) jobKey(id string) string    { return fmt.Sprintf("%s:job:%s", d.prefix, id) }
func (d *redisDriver) pendingKey() string        { return fmt.Sprintf("%s:queue:pending", d.prefix) }
func (d *redisDriver) deadKey() string           { return fmt.Sprintf("%s:queue:dead", d.prefix) }
func (d *redisDriver) cronKey() string           { return fmt.Sprintf("%s:crons", d.prefix) }

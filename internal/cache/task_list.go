package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/virtuxa/tz_task_manager/internal/task"
)

const taskCachePrefix = "task-manager:tasks"

type TaskListCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewTaskListCache(client *redis.Client, ttl time.Duration) *TaskListCache {
	return &TaskListCache{client: client, ttl: ttl}
}

func (cache *TaskListCache) Get(ctx context.Context, filter task.Filter) ([]task.Task, string, bool, error) {
	// Ищет список по ключу команды, фильтров и текущего поколения кеша
	version, err := cache.teamVersion(ctx, filter.TeamID)
	if err != nil {
		return nil, "", false, err
	}

	encodedTasks, err := cache.client.Get(ctx, cache.listKey(filter, version)).Bytes()
	if err == redis.Nil {
		return nil, version, false, nil
	}
	if err != nil {
		return nil, "", false, fmt.Errorf("read tasks cache: %w", err)
	}

	tasks := make([]task.Task, 0)
	if err := json.Unmarshal(encodedTasks, &tasks); err != nil {
		return nil, "", false, fmt.Errorf("decode tasks cache: %w", err)
	}

	return tasks, version, true, nil
}

func (cache *TaskListCache) Set(ctx context.Context, filter task.Filter, version string, tasks []task.Task) error {
	encodedTasks, err := json.Marshal(tasks)
	if err != nil {
		return fmt.Errorf("encode tasks cache: %w", err)
	}

	if err := cache.client.Set(ctx, cache.listKey(filter, version), encodedTasks, cache.ttl).Err(); err != nil {
		return fmt.Errorf("write tasks cache: %w", err)
	}

	return nil
}

func (cache *TaskListCache) InvalidateTeam(ctx context.Context, teamID int64) error {
	// Меняет поколение ключей, чтобы старые списки больше не использовались
	if err := cache.client.Incr(ctx, cache.versionKey(teamID)).Err(); err != nil {
		return fmt.Errorf("invalidate tasks cache: %w", err)
	}

	return nil
}

func (cache *TaskListCache) teamVersion(ctx context.Context, teamID int64) (string, error) {
	version, err := cache.client.Get(ctx, cache.versionKey(teamID)).Result()
	if err == redis.Nil {
		return "0", nil
	}
	if err != nil {
		return "", fmt.Errorf("read tasks cache version: %w", err)
	}

	return version, nil
}

func (cache *TaskListCache) versionKey(teamID int64) string {
	return taskCachePrefix + ":version:" + strconv.FormatInt(teamID, 10)
}

func (cache *TaskListCache) listKey(filter task.Filter, version string) string {
	status := "all"
	if filter.Status != nil {
		status = string(*filter.Status)
	}

	assigneeID := "all"
	if filter.AssigneeID != nil {
		assigneeID = strconv.FormatInt(*filter.AssigneeID, 10)
	}

	return fmt.Sprintf(
		"%s:list:%s:team:%d:status:%s:assignee:%s:limit:%d:offset:%d",
		taskCachePrefix,
		version,
		filter.TeamID,
		status,
		assigneeID,
		filter.Limit,
		filter.Offset,
	)
}

package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"
	"trainingApp/internal/stats"

	"github.com/redis/go-redis/v9"
)

type Dashboard struct {
	client *redis.Client
}

func NewDashboard(client *redis.Client) *Dashboard {
	return &Dashboard{
		client: client,
	}
}

func key(userID int64, from, to string) string {
	return fmt.Sprintf("stats:v1:%d:%s:%s",
		userID, url.QueryEscape(from), url.QueryEscape(to))
}

func (c *Dashboard) Get(ctx context.Context, userID int64, from, to string) (stats.Dashboard, bool, error) {
	data, err := c.client.Get(ctx, key(userID, from, to)).Bytes()
	if errors.Is(err, redis.Nil) {
		return stats.Dashboard{}, false, nil
	}
	if err != nil {
		return stats.Dashboard{}, false, err
	}
	var result stats.Dashboard
	if err := json.Unmarshal(data, &result); err != nil {
		return stats.Dashboard{}, false, err
	}
	return result, true, nil
}

func (c *Dashboard) Set(ctx context.Context, userID int64, from, to string, value stats.Dashboard) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key(userID, from, to), data, time.Minute).Err()
}

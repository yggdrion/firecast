package wiprecovery

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

func StartWipRecovery(ctx context.Context, rdb *redis.Client) {
	go func() {
		for {
			now := float64(time.Now().Unix())
			wipVideos, err := rdb.ZRangeByScoreWithScores(ctx, "videos:wip", &redis.ZRangeBy{
				Min: "-inf",
				Max: fmt.Sprintf("%f", now),
			}).Result()
			if err != nil {
				log.Printf("Error scanning videos:wip: %v", err)
				time.Sleep(10 * time.Second)
				continue
			}

			for _, z := range wipVideos {
				videoUuid := z.Member.(string)
				metaKey := fmt.Sprintf("videos:meta:%s", videoUuid)
				videoMeta, err := rdb.HGetAll(ctx, metaKey).Result()
				if err != nil {
					log.Printf("Error getting meta for %s: %v", videoUuid, err)
					continue
				}
				retries, _ := strconv.Atoi(videoMeta["retries"])

				if retries >= 3 {
					_, err := rdb.SAdd(ctx, "videos:failed", videoUuid).Result()
					if err != nil {
						log.Printf("Error moving %s to failed set: %v", videoUuid, err)
					}
				} else {
					_, err := rdb.LPush(ctx, "videos:queue", videoUuid).Result()
					if err != nil {
						log.Printf("Error moving %s back to queue: %v", videoUuid, err)
					}
					rdb.HSet(ctx, metaKey, "last_attempt_at", time.Now().Unix())
				}
				_, err = rdb.ZRem(ctx, "videos:wip", videoUuid).Result()
				if err != nil {
					log.Printf("Error removing %s from wip: %v", videoUuid, err)
				}
			}

			time.Sleep(10 * time.Second)
		}
	}()
}

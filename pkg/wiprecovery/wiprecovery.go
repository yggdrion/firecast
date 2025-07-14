package wiprecovery

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func WipRecovery(ctx context.Context, rdb *redis.Client) {

	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}

	wipTimeoutStr := os.Getenv("WIP_TIMEOUT")
	if wipTimeoutStr == "" {
		wipTimeoutStr = "300"
	}
	wipTimeout, err := strconv.Atoi(wipTimeoutStr)
	if err != nil {
		log.Printf("Invalid WIP_TIMEOUT value: %s, using default 300", wipTimeoutStr)
		wipTimeout = 300
	}

	wipRetryStr := os.Getenv("WIP_RETRY")
	if wipRetryStr == "" {
		wipRetryStr = "3"
	}
	maxRetries, err := strconv.Atoi(wipRetryStr)
	if err != nil {
		log.Printf("Invalid WIP_RETRY value: %s, using default 3", wipRetryStr)
		maxRetries = 3
	}

	wipFrequencyStr := os.Getenv("WIP_INTERVAL")
	if wipFrequencyStr == "" {
		wipFrequencyStr = "10"
	}
	wipFrequency, err := strconv.Atoi(wipFrequencyStr)
	if err != nil {
		log.Printf("Invalid WIP_INTERVAL value: %s, using default 10", wipFrequencyStr)
		wipFrequency = 10
	}

	go func() {
		for {
			now := float64(time.Now().Unix())
			timeoutThreshold := now - float64(wipTimeout)

			wipVideos, err := rdb.ZRangeByScoreWithScores(ctx, "videos:wip", &redis.ZRangeBy{
				Min: "-inf",
				Max: fmt.Sprintf("%f", timeoutThreshold),
			}).Result()
			if err != nil {
				log.Printf("Error scanning videos:wip: %v", err)
				time.Sleep(time.Duration(wipFrequency) * time.Second)
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

				_, err = rdb.ZRem(ctx, "videos:wip", videoUuid).Result()
				if err != nil {
					log.Printf("Error removing %s from wip: %v", videoUuid, err)
					continue
				}

				if retries >= maxRetries {
					_, err := rdb.SAdd(ctx, "videos:fail", videoUuid).Result()
					if err != nil {
						log.Printf("Error moving %s to fail set: %v", videoUuid, err)
					}
				} else {
					_, err := rdb.LPush(ctx, "videos:queue", videoUuid).Result()
					if err != nil {
						log.Printf("Error moving %s back to queue: %v", videoUuid, err)
					}
					rdb.HSet(ctx, metaKey, "last_attempt_at", time.Now().Unix())
				}
			}

			time.Sleep(time.Duration(wipFrequency) * time.Second)
		}
	}()
}

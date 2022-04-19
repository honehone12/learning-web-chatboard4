package session

import (
	"encoding/json"
	"learning-web-chatboard4/common/models"
	"time"

	"github.com/gomodule/redigo/redis"
)

const (
	redisConnectionKind = "tcp"
	redisAddress        = "localhost:6379"
)

// store session with expiration
func SetToRedisWithExpiration(sess *models.Session) (err error) {
	sess.LastUpdate = time.Now()
	bytes, err := json.Marshal(sess)
	if err != nil {
		return
	}
	r, err := sessionMaker.redisConn.Do(
		"SETEX",
		sess.UuId,
		sessionExpSec,
		bytes,
	)
	if err != nil {
		return
	}
	if sessionMaker.showRedisLog {
		sessionMaker.logger.Printf("SETEX: %v\n", r)
	}
	return
}

func SetToRedis(sess *models.Session) (err error) {
	sess.LastUpdate = time.Now()
	bytes, err := json.Marshal(sess)
	if err != nil {
		return
	}
	r, err := sessionMaker.redisConn.Do(
		"SET",
		sess.UuId,
		bytes,
	)
	if err != nil {
		return
	}
	if sessionMaker.showRedisLog {
		sessionMaker.logger.Printf("SET %v\n", r)
	}
	return
}

func ConfirmExistsFromRedis(uuid string) (exists bool, err error) {
	var r int64 = 0
	r, err = redis.Int64(
		sessionMaker.redisConn.Do("EXISTS", uuid),
	)
	exists = r == 1
	return
}

func GetFromRedis(uuid string, sess *models.Session) (err error) {
	bytes, err := redis.Bytes(
		sessionMaker.redisConn.Do("GET", uuid),
	)
	if err != nil {
		return
	}
	err = json.Unmarshal(bytes, sess)
	if err != nil {
		return
	}
	if sessionMaker.showRedisLog {
		sessionMaker.logger.Println(*sess)
	}
	return
}

func DelFromRedis(unique string) (err error) {
	r, err := sessionMaker.redisConn.Do("DEL", unique)
	if err != nil {
		return
	}
	if sessionMaker.showRedisLog {
		sessionMaker.logger.Printf("DEL %v\n", r)
	}
	return
}

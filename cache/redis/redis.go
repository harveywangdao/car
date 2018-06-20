package redis

import (
	"github.com/garyburd/redigo/redis"
	"github.com/harveywangdao/road/log/logger"
	"sync"
	"time"
)

type Redis struct {
	conn redis.Conn
}

const (
	MAX_POOL_SIZE  = 20
	MAX_IDLE_NUM   = 2
	MAX_ACTIVE_NUM = 20
	REDIS_ADDR     = "localhost:6379"
	REDISPASSWORD  = "180498"
)

var (
	lock      sync.Mutex
	redisPool *redis.Pool
)

func init() {
	redisPool = &redis.Pool{
		MaxIdle:     MAX_IDLE_NUM,
		MaxActive:   MAX_ACTIVE_NUM,
		IdleTimeout: (60 * time.Second),
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", REDIS_ADDR)
			if err != nil {
				logger.Error(err)
				return nil, err
			}

			if _, err := c.Do("AUTH", REDISPASSWORD); err != nil {
				c.Close()
				return nil, err
			}
			/*
				if _, err := c.Do("SELECT", db); err != nil {
					c.Close()
					return nil, err
				}*/
			return c, nil
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}
}

func NewRedis(ipport string) (*Redis, error) {
	red := &Redis{}
	red.conn = redisPool.Get()
	return red, nil
}

func (red *Redis) Close() {
	red.conn.Close()
}

func (red *Redis) IsKeyExist(key string) bool {
	exists, _ := redis.Bool(red.conn.Do("EXISTS", key))
	return exists
}

func (red *Redis) DeleteKey(key string) error {
	n, err := red.conn.Do("DEL", key)
	if err != nil {
		logger.Error(err, n)
		return err
	}

	return nil
}

func (red *Redis) CreateListByInt64Slice(key string, data []int64) error {
	//judge key exist
	if red.IsKeyExist(key) {
		logger.Warn(key, "Existed.")
		err := red.DeleteKey(key)
		if err != nil {
			logger.Error(err)
			return err
		}
	}

	//insert data
	for i := 0; i < len(data); i++ {
		_, err := red.conn.Do("RPUSH", key, data[i])
		if err != nil {
			logger.Error(err)
			return err
		}
	}

	return nil
}

func (red *Redis) GetListValueByIndex(key string, index int) (int64, error) {
	v, err := redis.Int64(red.conn.Do("LINDEX", key, index))
	if err != nil {
		logger.Error(err)
		return 0, err
	}

	return v, nil
}

func (red *Redis) GetListLen(key string) (int, error) {
	v, err := redis.Int(red.conn.Do("LLEN", key))
	if err != nil {
		logger.Error(err)
		return 0, err
	}

	return v, nil
}

func (red *Redis) GetInt64SliceList(key string) ([]int64, error) {
	l, err := red.GetListLen(key)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	v, err := redis.Int64s(red.conn.Do("LRANGE", key, 0, l))
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	return v, nil
}

func (red *Redis) SetListValueByIndex(key string, index int, value int64) error {
	_, err := red.conn.Do("LSET", key, index, value)
	if err != nil {
		logger.Error(err)
		return err
	}

	return nil
}

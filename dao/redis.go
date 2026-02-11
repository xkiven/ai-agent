package dao

import (
	"context"
	"encoding/json"
	"time"

	"ai-agent/model"
	"github.com/go-redis/redis/v8"
)

type RedisStore struct {
	client    *redis.Client
	keyPrefix string
	ttl       time.Duration
}

func NewRedisStore(addr, password string, db int) *RedisStore {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &RedisStore{
		client:    client,
		keyPrefix: "ai-agent:session:",
		ttl:       24 * time.Hour,
	}
}

func (s *RedisStore) Get(ctx context.Context, sessionID string) (*model.Session, error) {
	key := s.keyPrefix + sessionID
	data, err := s.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var session model.Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

func (s *RedisStore) Save(ctx context.Context, session *model.Session) error {
	key := s.keyPrefix + session.ID
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, key, data, s.ttl).Err()
}

func (s *RedisStore) Delete(ctx context.Context, sessionID string) error {
	key := s.keyPrefix + sessionID
	return s.client.Del(ctx, key).Err()
}

func (s *RedisStore) Close() error {
	return s.client.Close()
}

func (s *RedisStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

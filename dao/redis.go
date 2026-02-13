package dao

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"ai-agent/model"
	"github.com/go-redis/redis/v8"
)

// 定义错误类型
var (
	ErrSessionConflict = errors.New("session conflict: current session is newer")
	ErrMaxRetries      = errors.New("max retries exceeded")
	ErrInvalidSession  = errors.New("invalid session")
	ErrInvalidParam    = errors.New("invalid parameter")
)

type RedisStore struct {
	client    *redis.Client
	keyPrefix string
	ttl       time.Duration
}

func NewRedisStore(addr, password string, db int, ttl time.Duration) *RedisStore {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &RedisStore{
		client:    client,
		keyPrefix: "ai-agent:session:",
		ttl:       ttl,
	}
}

func (s *RedisStore) Get(ctx context.Context, sessionID string) (*model.Session, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("%w: sessionID is empty", ErrInvalidParam)
	}

	key := s.keyPrefix + sessionID
	data, err := s.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
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
	if err := s.validateSession(session); err != nil {
		return err
	}

	key := s.keyPrefix + session.ID
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, key, data, s.ttl).Err()
}

func (s *RedisStore) UpdateFlowState(ctx context.Context, sessionID string, step string, state map[string]interface{}) error {
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.CurrentStep = step
	session.FlowState = state
	session.UpdatedAt = time.Now().Format(time.RFC3339)

	return s.SaveWithOptimisticLock(ctx, session, 3)
}

// SaveWithOptimisticLock 使用乐观锁保存session，防止并发覆盖写
func (s *RedisStore) SaveWithOptimisticLock(ctx context.Context, session *model.Session, maxRetries int) error {
	// 参数校验
	if err := s.validateSession(session); err != nil {
		return err
	}
	if maxRetries < 0 {
		return fmt.Errorf("%w: maxRetries cannot be negative", ErrInvalidParam)
	}

	key := s.keyPrefix + session.ID

	for i := 0; i <= maxRetries; i++ {
		// 使用WATCH监控key
		err := s.client.Watch(ctx, func(tx *redis.Tx) error {
			// 获取当前session数据
			currentData, err := tx.Get(ctx, key).Bytes()
			if err != nil && !errors.Is(err, redis.Nil) {
				return err
			}

			// 如果session不存在，直接保存
			if errors.Is(err, redis.Nil) {
				// 设置初始版本号为1
				session.Version = 1
				data, err := json.Marshal(session)
				if err != nil {
					return err
				}
				return tx.Set(ctx, key, data, s.ttl).Err()
			}

			// 解析当前session数据
			var currentSession model.Session
			if err := json.Unmarshal(currentData, &currentSession); err != nil {
				return err
			}

			// 版本号检查：如果当前版本号大于等于我们要保存的版本号，说明有冲突
			if currentSession.Version > session.Version {
				return ErrSessionConflict
			}

			// 更新版本号：当前版本号+1
			session.Version = currentSession.Version + 1

			// 智能合并session数据
			mergedSession := s.mergeSessions(currentSession, *session)

			// 保存合并后的session
			data, err := json.Marshal(mergedSession)
			if err != nil {
				return err
			}

			// 直接使用Set命令
			return tx.Set(ctx, key, data, s.ttl).Err()
		}, key)

		// 检查错误类型，决定是否重试
		shouldRetry, retryErr := s.shouldRetry(err)
		if !shouldRetry {
			return retryErr
		}

		// 如果是可重试错误，等待后重试
		if i < maxRetries {
			time.Sleep(time.Millisecond * time.Duration(10*(i+1))) // 指数退避
			continue
		}

		return fmt.Errorf("%w for session %s: %v", ErrMaxRetries, session.ID, retryErr)
	}

	return fmt.Errorf("max retries exceeded for session %s", session.ID)
}

// validateSession 验证session参数
func (s *RedisStore) validateSession(session *model.Session) error {
	if session == nil {
		return fmt.Errorf("%w: session is nil", ErrInvalidSession)
	}
	if session.ID == "" {
		return fmt.Errorf("%w: session.ID is empty", ErrInvalidSession)
	}
	return nil
}

// shouldRetry 判断错误是否应该重试
func (s *RedisStore) shouldRetry(err error) (bool, error) {
	if err == nil {
		return false, nil
	}

	// Redis WATCH事务失败错误
	if errors.Is(err, redis.TxFailedErr) {
		return true, err
	}

	// 自定义的session冲突错误
	if errors.Is(err, ErrSessionConflict) {
		return true, err
	}

	// 其他错误不重试
	return false, err
}

// mergeSessions 智能合并两个session，保持消息顺序和状态一致性
func (s *RedisStore) mergeSessions(currentSession, newSession model.Session) model.Session {
	merged := currentSession

	// 1. 合并消息：按时间顺序合并，保持对话上下文
	merged.Messages = s.mergeMessages(currentSession.Messages, newSession.Messages)

	// 2. 状态合并：如果新状态比当前状态更高级，则使用新状态
	if s.isStateMoreAdvanced(newSession.State, currentSession.State) {
		merged.State = newSession.State
	}

	// 3. FlowID合并：优先使用新的FlowID
	if newSession.FlowID != "" {
		merged.FlowID = newSession.FlowID
	}
	// 4. CurrentStep 合并
	if newSession.CurrentStep != "" {
		merged.CurrentStep = newSession.CurrentStep
	}

	// 5. FlowState 合并
	if newSession.FlowState != nil {
		merged.FlowState = newSession.FlowState
	}

	return merged
}

// mergeMessages 按时间顺序合并消息，保持对话上下文
func (s *RedisStore) mergeMessages(currentMessages, newMessages []model.Message) []model.Message {
	// 使用版本号后，消息去重逻辑可以简化
	// 但为了保持消息顺序和上下文，我们仍然使用时间戳排序

	messageMap := make(map[string]model.Message)

	// 合并所有消息
	allMessages := append(currentMessages, newMessages...)

	for _, msg := range allMessages {
		msgID := s.generateMessageID(msg)
		messageMap[msgID] = msg
	}

	// 转换为切片
	result := make([]model.Message, 0, len(messageMap))
	for _, msg := range messageMap {
		result = append(result, msg)
	}

	// 按时间戳排序
	sort.Slice(result, func(i, j int) bool {
		return s.isTimestampNewer(result[j].Timestamp, result[i].Timestamp)
	})

	return result
}

// generateMessageID 生成消息的唯一ID
// 使用Role+Content+Timestamp作为ID，避免误判重复
// 使用Role+Content作为唯一标识符
func (s *RedisStore) generateMessageID(msg model.Message) string {
	return fmt.Sprintf("%s:%s", msg.Role, msg.Content)
}

// isTimestampNewer 安全比较时间戳，返回timestampA是否比timestampB更新
func (s *RedisStore) isTimestampNewer(timestampA, timestampB string) bool {
	// 尝试解析为RFC3339格式
	timeA, errA := time.Parse(time.RFC3339, timestampA)
	timeB, errB := time.Parse(time.RFC3339, timestampB)

	// 如果都能解析，直接比较时间
	if errA == nil && errB == nil {
		return timeA.After(timeB)
	}

	// 如果有一个解析失败，回退到字符串比较
	// 这比panic要好，但可能不准确
	return timestampA > timestampB
}

// isStateMoreAdvanced 检查状态stateA是否比stateB更"高级"
// 返回true表示stateA比stateB更高级
func (s *RedisStore) isStateMoreAdvanced(stateA, stateB model.SessionState) bool {
	stateOrder := map[model.SessionState]int{
		model.SessionNew:      0,
		model.SessionActive:   1,
		model.SessionOnFlow:   2,
		model.SessionComplete: 3,
	}

	orderA, existsA := stateOrder[stateA]
	orderB, existsB := stateOrder[stateB]

	// 如果状态未知，默认认为相等，不进行替换
	if !existsA || !existsB {
		return false
	}

	return orderA > orderB
}

func (s *RedisStore) Delete(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("%w: sessionID is empty", ErrInvalidParam)
	}

	key := s.keyPrefix + sessionID
	return s.client.Del(ctx, key).Err()
}

func (s *RedisStore) Close() error {
	return s.client.Close()
}

func (s *RedisStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

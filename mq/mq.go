package mq

import (
	"context"
	"time"
)

type Metadata struct {
	Partition       int
	Offset          int64
	DeliveryAttempt int
	ReceivedAt      time.Time
	Attributes      map[string]string
	Group           string
	ClientID        string
	SessionID       string
	MemberID        string
	GenerationID    int32
	PublishedAt     time.Time
	DeadLetter      *DeadLetterMetadata
}

type Message struct {
	ID       string
	Topic    string
	Key      string
	Body     []byte
	Headers  map[string]string
	Header   map[string]string
	Metadata Metadata
}

type Delivery struct {
	Message Message
	Ack     func(context.Context) error
	Nack    func(context.Context, error) error
	Decide  func(context.Context, Decision) error
}

type PublishResult struct {
	MessageID string
	Topic     string
	Partition int
	Offset    int64
	Timestamp time.Time
	Metadata  Metadata
}

type ProducerCallback interface {
	OnSuccess(ctx context.Context, message Message, result PublishResult)
	OnError(ctx context.Context, message Message, err error)
}

type ProducerCallbackFunc struct {
	Success func(context.Context, Message, PublishResult)
	Error   func(context.Context, Message, error)
}

func (fn ProducerCallbackFunc) OnSuccess(ctx context.Context, message Message, result PublishResult) {
	if fn.Success != nil {
		fn.Success(ctx, message, result)
	}
}

func (fn ProducerCallbackFunc) OnError(ctx context.Context, message Message, err error) {
	if fn.Error != nil {
		fn.Error(ctx, message, err)
	}
}

type Subscription struct {
	Group            string
	Topics           []string
	ClientID         string
	SessionID        string
	StartOffset      OffsetResetPolicy
	Retry            RetryPolicy
	DeadLetter       DeadLetterPolicy
	ErrorHandler     ErrorHandler
	MaxInFlight      int
	CommitOnDelivery bool
}

type Handler interface {
	HandleMessage(context.Context, Message) error
}

type HandlerFunc func(context.Context, Message) error

func (fn HandlerFunc) HandleMessage(ctx context.Context, message Message) error {
	return fn(ctx, message)
}

type Producer interface {
	Publish(ctx context.Context, message Message) error
}

type BatchProducer interface {
	PublishBatch(ctx context.Context, messages ...Message) ([]PublishResult, error)
}

type Consumer interface {
	Consume(ctx context.Context) (<-chan Delivery, error)
}

type GroupConsumer interface {
	Subscribe(ctx context.Context, subscription Subscription) (<-chan Delivery, error)
}

type ConsumerFunc func(context.Context) (<-chan Delivery, error)

func (fn ConsumerFunc) Consume(ctx context.Context) (<-chan Delivery, error) {
	return fn(ctx)
}

type GroupConsumerFunc func(context.Context, Subscription) (<-chan Delivery, error)

func (fn GroupConsumerFunc) Subscribe(ctx context.Context, subscription Subscription) (<-chan Delivery, error) {
	return fn(ctx, subscription)
}

type OffsetResetPolicy string

const (
	OffsetResetLatest   OffsetResetPolicy = "latest"
	OffsetResetEarliest OffsetResetPolicy = "earliest"
)

type Offset struct {
	Topic     string
	Partition int
	Offset    int64
	Metadata  string
	Timestamp time.Time
}

type Session interface {
	Commit(ctx context.Context, offsets ...Offset) error
}

type ErrorHandler interface {
	HandleConsumerError(ctx context.Context, err error, metadata Metadata)
}

type ErrorHandlerFunc func(context.Context, error, Metadata)

func (fn ErrorHandlerFunc) HandleConsumerError(ctx context.Context, err error, metadata Metadata) {
	fn(ctx, err, metadata)
}

type DeadLetterPolicy struct {
	Topic       string
	Reason      string
	MaxAttempts int
	Metadata    map[string]string
}

type DeadLetterMetadata struct {
	Topic             string
	Reason            string
	OriginalTopic     string
	OriginalGroup     string
	OriginalPartition int
	OriginalOffset    int64
	Attempts          int
	FailedAt          time.Time
	Attributes        map[string]string
}

type RetryPolicy struct {
	MaxAttempts    int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	Multiplier     float64
}

func (p RetryPolicy) ShouldRetry(attempt int, err error) bool {
	if err == nil {
		return false
	}
	if p.MaxAttempts <= 0 {
		return false
	}
	return attempt < p.MaxAttempts
}

func (p RetryPolicy) Backoff(attempt int) time.Duration {
	if attempt <= 0 || p.InitialBackoff <= 0 {
		return 0
	}
	backoff := p.InitialBackoff
	multiplier := p.Multiplier
	if multiplier < 1 {
		multiplier = 1
	}
	for i := 1; i < attempt; i++ {
		backoff = time.Duration(float64(backoff) * multiplier)
		if p.MaxBackoff > 0 && backoff > p.MaxBackoff {
			return p.MaxBackoff
		}
	}
	if p.MaxBackoff > 0 && backoff > p.MaxBackoff {
		return p.MaxBackoff
	}
	return backoff
}

type DecisionAction string

const (
	DecisionAck        DecisionAction = "ack"
	DecisionNack       DecisionAction = "nack"
	DecisionRetry      DecisionAction = "retry"
	DecisionDeadLetter DecisionAction = "dead_letter"
)

type Decision struct {
	Action     DecisionAction
	Cause      error
	RetryAfter time.Duration
	Metadata   map[string]string
	DeadLetter DeadLetterMetadata
}

func (d Delivery) AckMessage(ctx context.Context) error {
	if d.Decide != nil {
		return d.Decide(ctx, Decision{Action: DecisionAck})
	}
	if d.Ack == nil {
		return nil
	}
	return d.Ack(ctx)
}

func (d Delivery) NackMessage(ctx context.Context, cause error) error {
	if d.Decide != nil {
		return d.Decide(ctx, Decision{Action: DecisionNack, Cause: cause})
	}
	if d.Nack == nil {
		return nil
	}
	return d.Nack(ctx, cause)
}

func (d Delivery) RetryMessage(ctx context.Context, cause error, retryAfter time.Duration) error {
	if d.Decide != nil {
		return d.Decide(ctx, Decision{Action: DecisionRetry, Cause: cause, RetryAfter: retryAfter})
	}
	if d.Nack == nil {
		return nil
	}
	return d.Nack(ctx, cause)
}

func (d Delivery) DeadLetterMessage(ctx context.Context, cause error, metadata DeadLetterMetadata) error {
	if d.Decide != nil {
		return d.Decide(ctx, Decision{Action: DecisionDeadLetter, Cause: cause, DeadLetter: metadata})
	}
	if d.Nack == nil {
		return nil
	}
	return d.Nack(ctx, cause)
}

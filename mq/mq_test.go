package mq

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNoopMQImplementsProducerAndConsumer(t *testing.T) {
	noop := NewNoop()
	var producer Producer = noop
	var batchProducer BatchProducer = noop
	var consumer Consumer = noop
	var groupConsumer GroupConsumer = noop

	err := producer.Publish(context.Background(), Message{Topic: "events", Body: []byte("hello")})
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from Publish, got %v", err)
	}
	_, err = batchProducer.PublishBatch(context.Background(), Message{Topic: "events"})
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from PublishBatch, got %v", err)
	}

	_, err = consumer.Consume(context.Background())
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from Consume, got %v", err)
	}
	_, err = groupConsumer.Subscribe(context.Background(), Subscription{Group: "workers", Topics: []string{"events"}})
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from Subscribe, got %v", err)
	}
}

func TestMQOptions(t *testing.T) {
	callback := ProducerCallbackFunc{}
	handler := ErrorHandlerFunc(func(context.Context, error, Metadata) {})
	options := NewOptions(
		WithBroker("kafka"),
		WithTopic("events"),
		WithGroup("workers"),
		WithClientID("client-a"),
		WithSessionID("session-a"),
		WithProducerCallback(callback),
		WithRetryPolicy(RetryPolicy{MaxAttempts: 3, InitialBackoff: time.Millisecond}),
		WithDeadLetterPolicy(DeadLetterPolicy{Topic: "events.dlq"}),
		WithErrorHandler(handler),
	)
	if options.Broker != "kafka" {
		t.Fatalf("expected broker kafka, got %q", options.Broker)
	}
	if options.Topic != "events" {
		t.Fatalf("expected topic events, got %q", options.Topic)
	}
	if options.Group != "workers" || options.ClientID != "client-a" || options.SessionID != "session-a" {
		t.Fatalf("unexpected group options: %#v", options)
	}
	if options.Callback == nil || options.ErrorHandler == nil {
		t.Fatalf("expected callback and error handler configured: %#v", options)
	}
	if options.Retry.MaxAttempts != 3 || options.DeadLetter.Topic != "events.dlq" {
		t.Fatalf("unexpected reliability options: %#v", options)
	}
}

func TestRetryPolicyBackoff(t *testing.T) {
	policy := RetryPolicy{
		MaxAttempts:    4,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     25 * time.Millisecond,
		Multiplier:     2,
	}
	err := errors.New("temporary")
	if !policy.ShouldRetry(1, err) || !policy.ShouldRetry(3, err) {
		t.Fatal("expected attempts below max to retry")
	}
	if policy.ShouldRetry(4, err) || policy.ShouldRetry(1, nil) {
		t.Fatal("expected max attempt and nil error to stop retrying")
	}
	if got := policy.Backoff(1); got != 10*time.Millisecond {
		t.Fatalf("expected first backoff 10ms, got %v", got)
	}
	if got := policy.Backoff(3); got != 25*time.Millisecond {
		t.Fatalf("expected capped third backoff 25ms, got %v", got)
	}
}

func TestDeliveryDecisionHelpersPreferDecisionCallback(t *testing.T) {
	var decisions []Decision
	delivery := Delivery{
		Decide: func(ctx context.Context, decision Decision) error {
			decisions = append(decisions, decision)
			return nil
		},
		Ack: func(context.Context) error {
			t.Fatal("Ack should not be called when Decide is available")
			return nil
		},
		Nack: func(context.Context, error) error {
			t.Fatal("Nack should not be called when Decide is available")
			return nil
		},
	}

	cause := errors.New("failed")
	if err := delivery.AckMessage(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := delivery.RetryMessage(context.Background(), cause, time.Second); err != nil {
		t.Fatal(err)
	}
	if err := delivery.DeadLetterMessage(context.Background(), cause, DeadLetterMetadata{Topic: "events.dlq"}); err != nil {
		t.Fatal(err)
	}
	if len(decisions) != 3 {
		t.Fatalf("expected 3 decisions, got %d", len(decisions))
	}
	if decisions[0].Action != DecisionAck || decisions[1].Action != DecisionRetry || decisions[2].Action != DecisionDeadLetter {
		t.Fatalf("unexpected decisions: %#v", decisions)
	}
}

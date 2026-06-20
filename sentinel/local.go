package sentinel

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type RatePolicy struct {
	Resource string
	Limit    int
	Window   time.Duration
}

type BreakerPolicy struct {
	Resource         string
	FailureThreshold int
	SuccessThreshold int
	OpenTimeout      time.Duration
}

type ShedderPolicy struct {
	Resource     string
	MaxInFlight  int
	CPUThreshold float64
	MinRT        time.Duration
	Window       time.Duration
	Buckets      int
	CPU          func() float64
}

type PriorityPolicy struct {
	Resource    string
	MaxInFlight int
	MinPriority int
}

type WindowSample struct {
	Count         int64
	Failures      int64
	TotalDuration time.Duration
	MinDuration   time.Duration
}

type RollingWindow struct {
	mu      sync.Mutex
	window  time.Duration
	size    int
	now     func() time.Time
	buckets []windowBucket
}

type windowBucket struct {
	start         time.Time
	count         int64
	failures      int64
	totalDuration time.Duration
	minDuration   time.Duration
}

func NewRateLimiter(policy RatePolicy) Limiter {
	if policy.Window <= 0 {
		policy.Window = time.Second
	}
	return &rateLimiter{policy: policy, windows: map[string]rateWindow{}}
}

func NewCircuitBreaker(policy BreakerPolicy) Breaker {
	if policy.FailureThreshold <= 0 {
		policy.FailureThreshold = 5
	}
	if policy.SuccessThreshold <= 0 {
		policy.SuccessThreshold = 1
	}
	if policy.OpenTimeout <= 0 {
		policy.OpenTimeout = time.Second
	}
	return &circuitBreaker{policy: policy, state: map[string]breakerState{}}
}

func NewAdaptiveShedder(policy ShedderPolicy) Limiter {
	if policy.MaxInFlight <= 0 {
		policy.MaxInFlight = 1
	}
	if policy.CPUThreshold <= 0 {
		policy.CPUThreshold = 0.9
	}
	if policy.Window <= 0 {
		policy.Window = time.Second
	}
	if policy.Buckets <= 0 {
		policy.Buckets = 10
	}
	if policy.CPU == nil {
		policy.CPU = func() float64 { return 0 }
	}
	return &adaptiveShedder{
		policy: policy,
		window: NewRollingWindow(policy.Window, policy.Buckets),
	}
}

func NewPriorityShedder(policy PriorityPolicy) Limiter {
	if policy.MaxInFlight <= 0 {
		policy.MaxInFlight = 1
	}
	return &priorityShedder{policy: policy}
}

func NewRollingWindow(window time.Duration, buckets int) *RollingWindow {
	if window <= 0 {
		window = time.Second
	}
	if buckets <= 0 {
		buckets = 10
	}
	return &RollingWindow{
		window:  window,
		size:    buckets,
		now:     time.Now,
		buckets: make([]windowBucket, buckets),
	}
}

func (w *RollingWindow) Add(duration time.Duration, err error) {
	if w == nil {
		return
	}
	now := w.now()
	w.mu.Lock()
	defer w.mu.Unlock()
	bucket := w.bucket(now)
	bucket.count++
	if err != nil {
		bucket.failures++
	}
	bucket.totalDuration += duration
	if bucket.minDuration == 0 || (duration > 0 && duration < bucket.minDuration) {
		bucket.minDuration = duration
	}
}

func (w *RollingWindow) Snapshot() WindowSample {
	if w == nil {
		return WindowSample{}
	}
	now := w.now()
	w.mu.Lock()
	defer w.mu.Unlock()
	var sample WindowSample
	for _, bucket := range w.buckets {
		if bucket.start.IsZero() || now.Sub(bucket.start) > w.window {
			continue
		}
		sample.Count += bucket.count
		sample.Failures += bucket.failures
		sample.TotalDuration += bucket.totalDuration
		if sample.MinDuration == 0 || (bucket.minDuration > 0 && bucket.minDuration < sample.MinDuration) {
			sample.MinDuration = bucket.minDuration
		}
	}
	return sample
}

func (w *RollingWindow) bucket(now time.Time) *windowBucket {
	bucketDuration := w.window / time.Duration(w.size)
	if bucketDuration <= 0 {
		bucketDuration = w.window
	}
	start := now.Truncate(bucketDuration)
	index := int(start.UnixNano()/int64(bucketDuration)) % w.size
	if index < 0 {
		index = -index
	}
	bucket := &w.buckets[index]
	if !bucket.start.Equal(start) {
		*bucket = windowBucket{start: start}
	}
	return bucket
}

type rateLimiter struct {
	mu      sync.Mutex
	policy  RatePolicy
	windows map[string]rateWindow
}

type rateWindow struct {
	start time.Time
	count int
}

func (l *rateLimiter) Acquire(ctx context.Context, resource Resource) (Permit, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	key := resourceKey(l.policy.Resource, resource)
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	window := l.windows[key]
	if window.start.IsZero() || now.Sub(window.start) >= l.policy.Window {
		window = rateWindow{start: now}
	}
	if l.policy.Limit > 0 && window.count >= l.policy.Limit {
		l.windows[key] = window
		return nil, ErrRejected
	}
	window.count++
	l.windows[key] = window
	return permitFunc(func() {}), nil
}

type circuitBreaker struct {
	mu     sync.Mutex
	policy BreakerPolicy
	state  map[string]breakerState
}

type breakerState struct {
	openUntil time.Time
	failures  int
	successes int
	halfOpen  bool
}

func (b *circuitBreaker) Allow(ctx context.Context, resource Resource) (Guard, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	key := resourceKey(b.policy.Resource, resource)
	now := time.Now()
	b.mu.Lock()
	state := b.state[key]
	if !state.openUntil.IsZero() && now.Before(state.openUntil) && !state.halfOpen {
		b.mu.Unlock()
		return nil, ErrRejected
	}
	if !state.openUntil.IsZero() && !now.Before(state.openUntil) {
		state.halfOpen = true
		b.state[key] = state
	}
	b.mu.Unlock()
	return guardFunc(func(err error) { b.record(key, err) }), nil
}

func (b *circuitBreaker) record(key string, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	state := b.state[key]
	if err != nil {
		state.failures++
		state.successes = 0
		if state.failures >= b.policy.FailureThreshold || state.halfOpen {
			state.openUntil = time.Now().Add(b.policy.OpenTimeout)
			state.halfOpen = false
		}
		b.state[key] = state
		return
	}
	state.successes++
	if state.halfOpen && state.successes >= b.policy.SuccessThreshold {
		state = breakerState{}
	}
	state.failures = 0
	b.state[key] = state
}

type adaptiveShedder struct {
	policy   ShedderPolicy
	window   *RollingWindow
	inFlight atomic.Int64
}

type priorityShedder struct {
	policy   PriorityPolicy
	inFlight atomic.Int64
}

func (s *priorityShedder) Acquire(ctx context.Context, resource Resource) (Permit, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if configured := s.policy.Resource; configured != "" && configured != resource.Name {
		return permitFunc(func() {}), nil
	}
	if s.inFlight.Load() >= int64(s.policy.MaxInFlight) && resourcePriority(resource) < s.policy.MinPriority {
		return nil, ErrRejected
	}
	s.inFlight.Add(1)
	return permitFunc(func() { s.inFlight.Add(-1) }), nil
}

func resourcePriority(resource Resource) int {
	if resource.Priority != 0 {
		return resource.Priority
	}
	for _, attribute := range resource.Attributes {
		if attribute.Key != "priority" {
			continue
		}
		switch value := attribute.Value.(type) {
		case int:
			return value
		case int64:
			return int(value)
		case string:
			priority, err := strconv.Atoi(value)
			if err == nil {
				return priority
			}
		}
	}
	return 0
}

func (s *adaptiveShedder) Acquire(ctx context.Context, resource Resource) (Permit, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if configured := s.policy.Resource; configured != "" && configured != resource.Name {
		return permitFunc(func() {}), nil
	}
	inFlight := s.inFlight.Load()
	sample := s.window.Snapshot()
	if inFlight >= int64(s.policy.MaxInFlight) && s.policy.CPU() >= s.policy.CPUThreshold {
		if s.policy.MinRT <= 0 || sample.MinDuration == 0 || sample.MinDuration >= s.policy.MinRT {
			return nil, ErrRejected
		}
	}
	s.inFlight.Add(1)
	start := time.Now()
	return permitFunc(func() {
		s.inFlight.Add(-1)
		s.window.Add(time.Since(start), nil)
	}), nil
}

type permitFunc func()

func (fn permitFunc) Release() {
	if fn != nil {
		fn()
	}
}

type guardFunc func(error)

func (fn guardFunc) Done(err error) {
	if fn != nil {
		fn(err)
	}
}

func resourceKey(configured string, resource Resource) string {
	if configured != "" {
		return configured
	}
	if resource.Name != "" {
		return resource.Name
	}
	return "default"
}

package discovery

import (
	"context"
	"fmt"
	"hash/fnv"
	"math/rand"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type Service struct {
	Name      string
	Namespace string
	Group     string
	Metadata  map[string]string
}

type ServiceInstance struct {
	ID        string
	Name      string
	Version   string
	Namespace string
	Group     string
	Endpoints []Endpoint
	Metadata  map[string]string
	Topology  Topology
}

type TargetKind string

const (
	TargetKindDirect    TargetKind = "direct"
	TargetKindDiscovery TargetKind = "discovery"
)

const (
	TargetSchemeDirect    = "direct"
	TargetSchemeDiscovery = "discovery"
)

type Target struct {
	Kind     TargetKind
	Scheme   string
	Raw      string
	Service  string
	Endpoint Endpoint
	Metadata map[string]string
}

type Health int

const (
	HealthUnknown Health = iota
	HealthUnhealthy
	HealthDraining
	HealthServing
)

type Topology struct {
	Region  string
	Zone    string
	Cell    string
	Cluster string
	Node    string
}

type Endpoint struct {
	Scheme   string
	Network  string
	Addr     string
	Weight   int
	Health   Health
	Topology Topology
	Metadata map[string]string
}

type EndpointOption func(*Endpoint)

type Registrar interface {
	Register(ctx context.Context, instance ServiceInstance) error
	Deregister(ctx context.Context, instance ServiceInstance) error
}

type Resolver interface {
	Resolve(ctx context.Context, service string) ([]Endpoint, error)
}

type HealthChecker interface {
	Check(ctx context.Context) (ProviderHealth, error)
}

type MetadataProvider interface {
	Metadata() map[string]string
}

type Snapshotter interface {
	Snapshot(ctx context.Context, service string) (Snapshot, error)
}

type Watcher interface {
	Watch(ctx context.Context, service string) (<-chan Snapshot, error)
}

type Discovery interface {
	Resolver
	Snapshotter
	Watcher
	GetService(ctx context.Context, service string) ([]ServiceInstance, error)
}

type Provider interface {
	Resolver
	HealthChecker
	MetadataProvider
	Snapshotter
	Watcher
}

type SnapshotProvider interface {
	Provider
}

type ProviderHealth struct {
	Serving  bool
	Message  string
	Metadata map[string]string
}

type Snapshot struct {
	Service    string
	Descriptor Service
	Endpoints  []Endpoint
	Metadata   map[string]string
	Health     ProviderHealth
	Topology   Topology
}

type EndpointFilter func(Endpoint) bool

type EndpointOrder func([]Endpoint)

type BalanceStrategy string

const (
	BalanceFirst              BalanceStrategy = "first"
	BalanceWeighted           BalanceStrategy = "weighted"
	BalanceRandom             BalanceStrategy = "random"
	BalanceRoundRobin         BalanceStrategy = "round_robin"
	BalanceWeightedRoundRobin BalanceStrategy = "weighted_round_robin"
	BalanceP2C                BalanceStrategy = "p2c"
	BalanceEWMA               BalanceStrategy = "ewma"
	BalanceConsistentHash     BalanceStrategy = "consistent_hash"
)

type ResolvePolicy struct {
	Filters          []EndpointFilter
	Order            EndpointOrder
	Balancer         BalanceStrategy
	HashKey          string
	SubsetSize       int
	KeepStaleOnError bool
	StaleTTL         time.Duration
	Retry            RetryPolicy
}

type RetryPolicy struct {
	MaxAttempts       int
	InitialBackoff    time.Duration
	MaxBackoff        time.Duration
	BackoffMultiplier float64
}

type Selector interface {
	Select([]Endpoint) (Endpoint, bool)
	Report(Endpoint, time.Duration, error)
}

type SelectorOption func(*selector)

type StaticProviderOption func(*StaticProvider)

type StaticProvider struct {
	services    map[string][]Endpoint
	descriptors map[string]Service
	instances   map[string][]ServiceInstance
	metadata    map[string]string
	health      ProviderHealth
	topology    Topology
}

type StaticResolver map[string][]Endpoint

func (r StaticResolver) Resolve(ctx context.Context, service string) ([]Endpoint, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	endpoints := cloneEndpoints(r[service])
	if len(endpoints) == 0 {
		return nil, fmt.Errorf("discovery service has no endpoints: %s", service)
	}
	return endpoints, nil
}

func NewStaticProvider(services map[string][]Endpoint, opts ...StaticProviderOption) *StaticProvider {
	provider := &StaticProvider{
		services:    cloneServices(services),
		descriptors: make(map[string]Service),
		health: ProviderHealth{
			Serving: true,
		},
	}
	for _, opt := range opts {
		opt(provider)
	}
	return provider
}

func WithProviderInstances(instances []ServiceInstance) StaticProviderOption {
	return func(provider *StaticProvider) {
		provider.instances = cloneServiceInstanceMap(instances)
		if provider.services == nil {
			provider.services = map[string][]Endpoint{}
		}
		for service, values := range provider.instances {
			for _, instance := range values {
				provider.services[service] = append(provider.services[service], cloneEndpoints(instance.Endpoints)...)
			}
		}
	}
}

func WithProviderServices(services []Service) StaticProviderOption {
	return func(provider *StaticProvider) {
		provider.descriptors = make(map[string]Service, len(services))
		for _, service := range services {
			if service.Name == "" {
				continue
			}
			provider.descriptors[service.Name] = cloneService(service)
		}
	}
}

func WithProviderMetadata(metadata map[string]string) StaticProviderOption {
	return func(provider *StaticProvider) {
		provider.metadata = cloneMetadata(metadata)
	}
}

func WithProviderHealth(health ProviderHealth) StaticProviderOption {
	return func(provider *StaticProvider) {
		health.Metadata = cloneMetadata(health.Metadata)
		provider.health = health
	}
}

func WithProviderTopology(topology Topology) StaticProviderOption {
	return func(provider *StaticProvider) {
		provider.topology = topology
	}
}

func (p *StaticProvider) Resolve(ctx context.Context, service string) ([]Endpoint, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	endpoints := cloneEndpoints(p.services[service])
	if len(endpoints) == 0 {
		return nil, fmt.Errorf("discovery service has no endpoints: %s", service)
	}
	return endpoints, nil
}

func (p *StaticProvider) Check(ctx context.Context) (ProviderHealth, error) {
	if err := ctx.Err(); err != nil {
		return ProviderHealth{}, err
	}
	health := p.health
	if health.Metadata == nil {
		health.Metadata = cloneMetadata(p.metadata)
	} else {
		health.Metadata = cloneMetadata(health.Metadata)
	}
	return health, nil
}

func (p *StaticProvider) Metadata() map[string]string {
	return cloneMetadata(p.metadata)
}

func (p *StaticProvider) GetService(ctx context.Context, service string) ([]ServiceInstance, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if instances := cloneServiceInstances(p.instances[service]); len(instances) > 0 {
		return instances, nil
	}
	endpoints, err := p.Resolve(ctx, service)
	if err != nil {
		return nil, err
	}
	descriptor := p.descriptors[service]
	if descriptor.Name == "" {
		descriptor.Name = service
	}
	return []ServiceInstance{{
		ID:        descriptor.Name,
		Name:      descriptor.Name,
		Namespace: descriptor.Namespace,
		Group:     descriptor.Group,
		Endpoints: endpoints,
		Metadata:  cloneMetadata(descriptor.Metadata),
		Topology:  p.topology,
	}}, nil
}

func (p *StaticProvider) Snapshot(ctx context.Context, service string) (Snapshot, error) {
	endpoints, err := p.Resolve(ctx, service)
	if err != nil {
		return Snapshot{}, err
	}
	health, err := p.Check(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	descriptor := p.descriptors[service]
	if descriptor.Name == "" {
		descriptor.Name = service
	}
	return Snapshot{
		Service:    service,
		Descriptor: cloneService(descriptor),
		Endpoints:  endpoints,
		Metadata:   p.Metadata(),
		Health:     health,
		Topology:   p.topology,
	}, nil
}

func (p *StaticProvider) Watch(ctx context.Context, service string) (<-chan Snapshot, error) {
	snapshot, err := p.Snapshot(ctx, service)
	if err != nil {
		return nil, err
	}
	updates := make(chan Snapshot, 1)
	select {
	case <-ctx.Done():
		close(updates)
		return updates, nil
	case updates <- snapshot:
		close(updates)
		return updates, nil
	}
}

func ResolveWithPolicy(ctx context.Context, resolver Resolver, service string, policy ResolvePolicy) ([]Endpoint, error) {
	endpoints, err := resolver.Resolve(ctx, service)
	if err != nil {
		return nil, err
	}
	endpoints = ApplyResolvePolicy(endpoints, policy)
	if len(endpoints) == 0 {
		return nil, fmt.Errorf("discovery service has no endpoints after policy: %s", service)
	}
	return endpoints, nil
}

func SnapshotWithPolicy(ctx context.Context, provider SnapshotProvider, service string, policy ResolvePolicy) (Snapshot, error) {
	snapshot, err := provider.Snapshot(ctx, service)
	if err != nil {
		return Snapshot{}, err
	}
	snapshot.Endpoints = ApplyResolvePolicy(snapshot.Endpoints, policy)
	if len(snapshot.Endpoints) == 0 {
		return Snapshot{}, fmt.Errorf("discovery service has no endpoints after policy: %s", service)
	}
	return snapshot, nil
}

func WatchWithPolicy(ctx context.Context, provider SnapshotProvider, service string, policy ResolvePolicy) (<-chan Snapshot, error) {
	var updates <-chan Snapshot
	err := Retry(ctx, policy.Retry, func(ctx context.Context) error {
		next, err := provider.Watch(ctx, service)
		if err != nil {
			return err
		}
		updates = next
		return nil
	})
	if err != nil {
		if policy.KeepStaleOnError {
			return staleSnapshotChannel(ctx, provider, service, policy)
		}
		return nil, err
	}
	filtered := make(chan Snapshot, 1)
	go func() {
		defer close(filtered)
		var last Snapshot
		var lastAt time.Time
		for snapshot := range updates {
			snapshot.Endpoints = ApplyResolvePolicy(snapshot.Endpoints, policy)
			if len(snapshot.Endpoints) == 0 {
				if policy.KeepStaleOnError && len(last.Endpoints) > 0 && staleAllowed(lastAt, policy.StaleTTL) {
					select {
					case <-ctx.Done():
						return
					case filtered <- cloneSnapshot(last):
					}
				}
				continue
			}
			last = cloneSnapshot(snapshot)
			lastAt = time.Now()
			select {
			case <-ctx.Done():
				return
			case filtered <- snapshot:
			}
		}
	}()
	return filtered, nil
}

func ApplyResolvePolicy(endpoints []Endpoint, policy ResolvePolicy) []Endpoint {
	filtered := FilterEndpoints(endpoints, policy.Filters...)
	filtered = applySubset(filtered, policy)
	if policy.Order != nil {
		policy.Order(filtered)
	}
	return filtered
}

func FilterEndpoints(endpoints []Endpoint, filters ...EndpointFilter) []Endpoint {
	filtered := make([]Endpoint, 0, len(endpoints))
	for _, endpoint := range endpoints {
		if endpointMatches(endpoint, filters...) {
			filtered = append(filtered, Endpoint{
				Scheme:   endpoint.Scheme,
				Network:  endpoint.Network,
				Addr:     endpoint.Addr,
				Weight:   endpoint.Weight,
				Health:   endpoint.Health,
				Topology: endpoint.Topology,
				Metadata: cloneMetadata(endpoint.Metadata),
			})
		}
	}
	return filtered
}

func DirectEndpoint(address string, options ...EndpointOption) Endpoint {
	endpoint := Endpoint{
		Scheme: TargetSchemeDirect,
		Addr:   strings.TrimSpace(address),
	}
	for _, option := range options {
		if option != nil {
			option(&endpoint)
		}
	}
	endpoint.Metadata = cloneMetadata(endpoint.Metadata)
	return endpoint
}

func WithEndpointScheme(scheme string) EndpointOption {
	return func(endpoint *Endpoint) {
		endpoint.Scheme = strings.TrimSpace(scheme)
	}
}

func WithEndpointNetwork(network string) EndpointOption {
	return func(endpoint *Endpoint) {
		endpoint.Network = strings.TrimSpace(network)
	}
}

func WithEndpointWeight(weight int) EndpointOption {
	return func(endpoint *Endpoint) {
		endpoint.Weight = weight
	}
}

func WithEndpointHealth(health Health) EndpointOption {
	return func(endpoint *Endpoint) {
		endpoint.Health = health
	}
}

func WithEndpointTopology(topology Topology) EndpointOption {
	return func(endpoint *Endpoint) {
		endpoint.Topology = topology
	}
}

func WithEndpointMetadata(key string, value string) EndpointOption {
	return func(endpoint *Endpoint) {
		if strings.TrimSpace(key) == "" {
			return
		}
		if endpoint.Metadata == nil {
			endpoint.Metadata = map[string]string{}
		}
		endpoint.Metadata[key] = value
	}
}

func WithEndpointMetadataMap(metadata map[string]string) EndpointOption {
	return func(endpoint *Endpoint) {
		endpoint.Metadata = mergeMetadata(endpoint.Metadata, metadata)
	}
}

func DirectTarget(address string) string {
	return strings.TrimSpace(address)
}

func DiscoveryTarget(service string) string {
	return TargetSchemeDiscovery + ":///" + strings.Trim(strings.TrimSpace(service), "/")
}

func ParseTarget(raw string) (Target, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Target{}, fmt.Errorf("discovery target is empty")
	}
	if !strings.Contains(raw, "://") {
		return Target{
			Kind:   TargetKindDirect,
			Scheme: TargetSchemeDirect,
			Raw:    raw,
			Endpoint: Endpoint{
				Scheme: TargetSchemeDirect,
				Addr:   raw,
			},
		}, nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return Target{}, fmt.Errorf("parse discovery target %q: %w", raw, err)
	}
	scheme := strings.ToLower(parsed.Scheme)
	value := targetValue(parsed)
	switch scheme {
	case TargetSchemeDiscovery:
		if value == "" {
			return Target{}, fmt.Errorf("discovery target service is empty")
		}
		return Target{
			Kind:    TargetKindDiscovery,
			Scheme:  scheme,
			Raw:     raw,
			Service: value,
		}, nil
	case TargetSchemeDirect:
		if value == "" {
			return Target{}, fmt.Errorf("direct target address is empty")
		}
		return Target{
			Kind:   TargetKindDirect,
			Scheme: scheme,
			Raw:    raw,
			Endpoint: Endpoint{
				Scheme: scheme,
				Addr:   value,
			},
		}, nil
	default:
		return Target{
			Kind:   TargetKindDirect,
			Scheme: scheme,
			Raw:    raw,
			Endpoint: Endpoint{
				Scheme: scheme,
				Addr:   raw,
			},
		}, nil
	}
}

func EndpointMetadataEquals(key string, value string) EndpointFilter {
	return func(endpoint Endpoint) bool {
		return endpoint.Metadata[key] == value
	}
}

func EndpointWeightAtLeast(min int) EndpointFilter {
	return func(endpoint Endpoint) bool {
		return endpoint.Weight >= min
	}
}

func EndpointHealthAtLeast(min Health) EndpointFilter {
	return func(endpoint Endpoint) bool {
		return endpoint.Health >= min
	}
}

func EndpointInTopology(topology Topology) EndpointFilter {
	return func(endpoint Endpoint) bool {
		return topologyMatches(endpoint.Topology, topology)
	}
}

func EndpointVersion(version string) EndpointFilter {
	return func(endpoint Endpoint) bool {
		version = strings.TrimSpace(version)
		return version == "" ||
			endpoint.Metadata["version"] == version ||
			endpoint.Metadata["nucleus.version"] == version
	}
}

func OrderByWeightDesc(endpoints []Endpoint) {
	sort.SliceStable(endpoints, func(i, j int) bool {
		if endpoints[i].Weight == endpoints[j].Weight {
			return endpoints[i].Addr < endpoints[j].Addr
		}
		return endpoints[i].Weight > endpoints[j].Weight
	})
}

func OrderByWeightAsc(endpoints []Endpoint) {
	sort.SliceStable(endpoints, func(i, j int) bool {
		if endpoints[i].Weight == endpoints[j].Weight {
			return endpoints[i].Addr < endpoints[j].Addr
		}
		return endpoints[i].Weight < endpoints[j].Weight
	})
}

func SelectEndpoint(endpoints []Endpoint, strategy BalanceStrategy) (Endpoint, bool) {
	if len(endpoints) == 0 {
		return Endpoint{}, false
	}
	switch strategy {
	case BalanceWeighted:
		selected := endpoints[0]
		for _, endpoint := range endpoints[1:] {
			if endpoint.Weight > selected.Weight {
				selected = endpoint
			}
		}
		return selected, true
	case BalanceRandom:
		return endpoints[rand.Intn(len(endpoints))], true
	case BalanceP2C, BalanceEWMA:
		return selectPowerOfTwo(endpoints), true
	case BalanceConsistentHash:
		return selectConsistentHash(endpoints, ""), true
	default:
		return endpoints[0], true
	}
}

func NewSelector(policy ResolvePolicy, options ...SelectorOption) Selector {
	s := &selector{
		policy:  policy,
		current: map[string]int{},
		ewma:    map[string]float64{},
		rand:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	for _, option := range options {
		if option != nil {
			option(s)
		}
	}
	return s
}

func WithSelectorRand(random *rand.Rand) SelectorOption {
	return func(s *selector) {
		if random != nil {
			s.rand = random
		}
	}
}

type selector struct {
	mu      sync.Mutex
	policy  ResolvePolicy
	round   int
	current map[string]int
	ewma    map[string]float64
	rand    *rand.Rand
}

func (s *selector) Select(endpoints []Endpoint) (Endpoint, bool) {
	filtered := ApplyResolvePolicy(endpoints, s.policy)
	if len(filtered) == 0 {
		return Endpoint{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	switch s.policy.Balancer {
	case BalanceRandom:
		return filtered[s.rand.Intn(len(filtered))], true
	case BalanceRoundRobin:
		endpoint := filtered[s.round%len(filtered)]
		s.round++
		return endpoint, true
	case BalanceWeightedRoundRobin:
		return s.weightedRoundRobin(filtered), true
	case BalanceP2C:
		return s.powerOfTwo(filtered, false), true
	case BalanceEWMA:
		return s.powerOfTwo(filtered, true), true
	case BalanceConsistentHash:
		return selectConsistentHash(filtered, s.policy.HashKey), true
	default:
		return SelectEndpoint(filtered, s.policy.Balancer)
	}
}

func Retry(ctx context.Context, policy RetryPolicy, operation func(context.Context) error) error {
	if operation == nil {
		return nil
	}
	attempts := policy.MaxAttempts
	if attempts <= 0 {
		attempts = 1
	}
	delay := policy.InitialBackoff
	if delay < 0 {
		delay = 0
	}
	var err error
	for attempt := 1; attempt <= attempts; attempt++ {
		if err = ctx.Err(); err != nil {
			return err
		}
		if err = operation(ctx); err == nil {
			return nil
		}
		if attempt == attempts {
			return err
		}
		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
		delay = nextBackoff(delay, policy)
	}
	return err
}

func (s *selector) Report(endpoint Endpoint, duration time.Duration, err error) {
	addr := strings.TrimSpace(endpoint.Addr)
	if addr == "" {
		return
	}
	value := float64(duration.Microseconds()) / 1000
	if err != nil {
		value += 10000
	}
	if value <= 0 {
		value = 1
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.ewma[addr]
	if previous <= 0 {
		s.ewma[addr] = value
		return
	}
	s.ewma[addr] = previous*0.8 + value*0.2
}

func (s *selector) weightedRoundRobin(endpoints []Endpoint) Endpoint {
	total := 0
	var selected Endpoint
	selectedKey := ""
	selectedScore := -1
	for _, endpoint := range endpoints {
		weight := endpointWeight(endpoint)
		total += weight
		key := endpointKey(endpoint)
		s.current[key] += weight
		if selectedKey == "" || s.current[key] > selectedScore {
			selected = endpoint
			selectedKey = key
			selectedScore = s.current[key]
		}
	}
	if selectedKey != "" {
		s.current[selectedKey] -= total
	}
	return selected
}

func (s *selector) powerOfTwo(endpoints []Endpoint, useEWMA bool) Endpoint {
	if len(endpoints) == 1 {
		return endpoints[0]
	}
	first := endpoints[s.rand.Intn(len(endpoints))]
	second := endpoints[s.rand.Intn(len(endpoints))]
	for len(endpoints) > 1 && endpointKey(first) == endpointKey(second) {
		second = endpoints[s.rand.Intn(len(endpoints))]
	}
	if endpointScore(first, s.ewma, useEWMA) <= endpointScore(second, s.ewma, useEWMA) {
		return first
	}
	return second
}

func staleSnapshotChannel(ctx context.Context, provider SnapshotProvider, service string, policy ResolvePolicy) (<-chan Snapshot, error) {
	snapshot, err := SnapshotWithPolicy(ctx, provider, service, policy)
	if err != nil {
		return nil, err
	}
	updates := make(chan Snapshot, 1)
	select {
	case <-ctx.Done():
		close(updates)
	case updates <- snapshot:
		close(updates)
	}
	return updates, nil
}

func staleAllowed(lastAt time.Time, ttl time.Duration) bool {
	return !lastAt.IsZero() && (ttl <= 0 || time.Since(lastAt) <= ttl)
}

func selectPowerOfTwo(endpoints []Endpoint) Endpoint {
	if len(endpoints) == 1 {
		return endpoints[0]
	}
	first := endpoints[rand.Intn(len(endpoints))]
	second := endpoints[rand.Intn(len(endpoints))]
	if endpointScore(first, nil, false) <= endpointScore(second, nil, false) {
		return first
	}
	return second
}

func selectConsistentHash(endpoints []Endpoint, key string) Endpoint {
	selected := endpoints[0]
	selectedScore := hashScore(key, endpointKey(selected))
	for _, endpoint := range endpoints[1:] {
		score := hashScore(key, endpointKey(endpoint))
		if score > selectedScore || (score == selectedScore && endpointKey(endpoint) < endpointKey(selected)) {
			selected = endpoint
			selectedScore = score
		}
	}
	return selected
}

func applySubset(endpoints []Endpoint, policy ResolvePolicy) []Endpoint {
	if policy.SubsetSize <= 0 || policy.SubsetSize >= len(endpoints) {
		return endpoints
	}
	type candidate struct {
		index int
		score uint64
	}
	candidates := make([]candidate, 0, len(endpoints))
	for index, endpoint := range endpoints {
		candidates = append(candidates, candidate{
			index: index,
			score: hashScore(policy.HashKey, endpointKey(endpoint)),
		})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return endpointKey(endpoints[candidates[i].index]) < endpointKey(endpoints[candidates[j].index])
		}
		return candidates[i].score > candidates[j].score
	})
	selected := make(map[int]bool, policy.SubsetSize)
	for _, candidate := range candidates[:policy.SubsetSize] {
		selected[candidate.index] = true
	}
	subset := make([]Endpoint, 0, policy.SubsetSize)
	for index, endpoint := range endpoints {
		if selected[index] {
			subset = append(subset, endpoint)
		}
	}
	return subset
}

func nextBackoff(current time.Duration, policy RetryPolicy) time.Duration {
	if current <= 0 {
		current = policy.InitialBackoff
	}
	if current <= 0 {
		current = 50 * time.Millisecond
	}
	multiplier := policy.BackoffMultiplier
	if multiplier <= 0 {
		multiplier = 2
	}
	next := time.Duration(float64(current) * multiplier)
	if next <= current {
		next = current
	}
	if policy.MaxBackoff > 0 && next > policy.MaxBackoff {
		return policy.MaxBackoff
	}
	return next
}

func hashScore(values ...string) uint64 {
	hash := fnv.New64a()
	for _, value := range values {
		_, _ = hash.Write([]byte(value))
		_, _ = hash.Write([]byte{0})
	}
	return hash.Sum64()
}

func endpointScore(endpoint Endpoint, ewma map[string]float64, useEWMA bool) float64 {
	score := 1000.0 / float64(endpointWeight(endpoint))
	if useEWMA {
		if value := ewma[endpointKey(endpoint)]; value > 0 {
			score = value / float64(endpointWeight(endpoint))
		}
	}
	if endpoint.Health == HealthUnhealthy {
		score += 100000
	}
	if endpoint.Health == HealthDraining {
		score += 10000
	}
	if inflight, ok := endpoint.Metadata["inflight"]; ok {
		var value int
		if _, err := fmt.Sscanf(inflight, "%d", &value); err == nil && value > 0 {
			score += float64(value) * 100
		}
	}
	return score
}

func endpointWeight(endpoint Endpoint) int {
	if endpoint.Weight <= 0 {
		return 1
	}
	return endpoint.Weight
}

func endpointKey(endpoint Endpoint) string {
	if endpoint.Addr != "" {
		return endpoint.Addr
	}
	return endpoint.Scheme + "://" + endpoint.Network
}

func cloneServices(services map[string][]Endpoint) map[string][]Endpoint {
	cloned := make(map[string][]Endpoint, len(services))
	for service, endpoints := range services {
		cloned[service] = cloneEndpoints(endpoints)
	}
	return cloned
}

func cloneEndpoints(endpoints []Endpoint) []Endpoint {
	cloned := make([]Endpoint, len(endpoints))
	for i, endpoint := range endpoints {
		cloned[i] = Endpoint{
			Scheme:   endpoint.Scheme,
			Network:  endpoint.Network,
			Addr:     endpoint.Addr,
			Weight:   endpoint.Weight,
			Health:   endpoint.Health,
			Topology: endpoint.Topology,
			Metadata: cloneMetadata(endpoint.Metadata),
		}
	}
	return cloned
}

func cloneService(service Service) Service {
	service.Metadata = cloneMetadata(service.Metadata)
	return service
}

func cloneSnapshot(snapshot Snapshot) Snapshot {
	snapshot.Descriptor = cloneService(snapshot.Descriptor)
	snapshot.Endpoints = cloneEndpoints(snapshot.Endpoints)
	snapshot.Metadata = cloneMetadata(snapshot.Metadata)
	snapshot.Health.Metadata = cloneMetadata(snapshot.Health.Metadata)
	return snapshot
}

func cloneServiceInstance(instance ServiceInstance) ServiceInstance {
	instance.Endpoints = cloneEndpoints(instance.Endpoints)
	instance.Metadata = cloneMetadata(instance.Metadata)
	return instance
}

func cloneServiceInstances(instances []ServiceInstance) []ServiceInstance {
	if len(instances) == 0 {
		return nil
	}
	cloned := make([]ServiceInstance, len(instances))
	for i, instance := range instances {
		cloned[i] = cloneServiceInstance(instance)
	}
	return cloned
}

func cloneServiceInstanceMap(instances []ServiceInstance) map[string][]ServiceInstance {
	if len(instances) == 0 {
		return nil
	}
	cloned := map[string][]ServiceInstance{}
	for _, instance := range instances {
		if strings.TrimSpace(instance.Name) == "" {
			continue
		}
		cloned[instance.Name] = append(cloned[instance.Name], cloneServiceInstance(instance))
	}
	if len(cloned) == 0 {
		return nil
	}
	return cloned
}

func endpointMatches(endpoint Endpoint, filters ...EndpointFilter) bool {
	for _, filter := range filters {
		if filter != nil && !filter(endpoint) {
			return false
		}
	}
	return true
}

func topologyMatches(endpoint Topology, filter Topology) bool {
	if filter.Region != "" && endpoint.Region != filter.Region {
		return false
	}
	if filter.Zone != "" && endpoint.Zone != filter.Zone {
		return false
	}
	if filter.Cell != "" && endpoint.Cell != filter.Cell {
		return false
	}
	if filter.Cluster != "" && endpoint.Cluster != filter.Cluster {
		return false
	}
	if filter.Node != "" && endpoint.Node != filter.Node {
		return false
	}
	return true
}

func targetValue(parsed *url.URL) string {
	values := make([]string, 0, 2)
	if parsed.Host != "" {
		values = append(values, parsed.Host)
	}
	if path := strings.Trim(parsed.Path, "/"); path != "" {
		values = append(values, path)
	}
	return strings.Join(values, "/")
}

func cloneMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}
	return cloned
}

func mergeMetadata(values ...map[string]string) map[string]string {
	var merged map[string]string
	for _, value := range values {
		for key, item := range value {
			if strings.TrimSpace(key) == "" {
				continue
			}
			if merged == nil {
				merged = map[string]string{}
			}
			merged[key] = item
		}
	}
	return merged
}

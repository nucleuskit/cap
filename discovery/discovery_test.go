package discovery

import (
	"context"
	"errors"
	"math/rand"
	"testing"
	"time"
)

func TestParseTargetSupportsDirectAndDiscoveryTargets(t *testing.T) {
	tests := []struct {
		name        string
		raw         string
		wantKind    TargetKind
		wantService string
		wantAddr    string
	}{
		{
			name:     "plain address is direct",
			raw:      "127.0.0.1:50051",
			wantKind: TargetKindDirect,
			wantAddr: "127.0.0.1:50051",
		},
		{
			name:     "direct scheme is direct",
			raw:      "direct:///127.0.0.1:50052",
			wantKind: TargetKindDirect,
			wantAddr: "127.0.0.1:50052",
		},
		{
			name:        "discovery scheme uses path service",
			raw:         "discovery:///checkout",
			wantKind:    TargetKindDiscovery,
			wantService: "checkout",
		},
		{
			name:        "discovery scheme uses host service",
			raw:         "discovery://checkout",
			wantKind:    TargetKindDiscovery,
			wantService: "checkout",
		},
		{
			name:     "grpc native scheme remains direct target",
			raw:      "dns:///checkout.internal:50051",
			wantKind: TargetKindDirect,
			wantAddr: "dns:///checkout.internal:50051",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target, err := ParseTarget(tt.raw)
			if err != nil {
				t.Fatal(err)
			}
			if target.Kind != tt.wantKind {
				t.Fatalf("kind = %q, want %q", target.Kind, tt.wantKind)
			}
			if target.Service != tt.wantService {
				t.Fatalf("service = %q, want %q", target.Service, tt.wantService)
			}
			if target.Endpoint.Addr != tt.wantAddr {
				t.Fatalf("addr = %q, want %q", target.Endpoint.Addr, tt.wantAddr)
			}
		})
	}
}

func TestParseTargetRejectsEmptyDiscoveryService(t *testing.T) {
	if _, err := ParseTarget("discovery:///"); err == nil {
		t.Fatal("expected empty discovery service error")
	}
	if _, err := ParseTarget(" "); err == nil {
		t.Fatal("expected empty target error")
	}
}

func TestDirectAndDiscoveryTargetHelpers(t *testing.T) {
	endpoint := DirectEndpoint("127.0.0.1:50051",
		WithEndpointWeight(3),
		WithEndpointHealth(HealthServing),
		WithEndpointMetadata("zone", "dev-a"),
	)
	if endpoint.Addr != "127.0.0.1:50051" || endpoint.Weight != 3 || endpoint.Health != HealthServing {
		t.Fatalf("unexpected endpoint: %#v", endpoint)
	}
	if endpoint.Metadata["zone"] != "dev-a" {
		t.Fatalf("unexpected endpoint metadata: %#v", endpoint.Metadata)
	}
	if got := DirectTarget("127.0.0.1:50051"); got != "127.0.0.1:50051" {
		t.Fatalf("direct target = %q", got)
	}
	if got := DiscoveryTarget("checkout"); got != "discovery:///checkout" {
		t.Fatalf("discovery target = %q", got)
	}
}

func TestStaticProviderGetServiceSynthesizesAndClonesInstances(t *testing.T) {
	provider := NewStaticProvider(map[string][]Endpoint{
		"checkout": {{
			Addr:     "127.0.0.1:50051",
			Weight:   10,
			Health:   HealthServing,
			Metadata: map[string]string{"zone": "dev-a"},
		}},
	}, WithProviderServices([]Service{{
		Name:      "checkout",
		Namespace: "public",
		Group:     "DEFAULT_GROUP",
		Metadata:  map[string]string{"owner": "platform"},
	}}))

	var _ Discovery = provider

	instances, err := provider.GetService(context.Background(), "checkout")
	if err != nil {
		t.Fatal(err)
	}
	if len(instances) != 1 {
		t.Fatalf("expected one service instance, got %#v", instances)
	}
	instance := instances[0]
	if instance.Name != "checkout" || instance.Namespace != "public" || instance.Group != "DEFAULT_GROUP" {
		t.Fatalf("unexpected instance identity: %#v", instance)
	}
	if instance.Metadata["owner"] != "platform" {
		t.Fatalf("unexpected instance metadata: %#v", instance.Metadata)
	}
	if len(instance.Endpoints) != 1 || instance.Endpoints[0].Metadata["zone"] != "dev-a" {
		t.Fatalf("unexpected instance endpoints: %#v", instance.Endpoints)
	}

	instances[0].Metadata["owner"] = "mutated"
	instances[0].Endpoints[0].Metadata["zone"] = "mutated"

	next, err := provider.GetService(context.Background(), "checkout")
	if err != nil {
		t.Fatal(err)
	}
	if next[0].Metadata["owner"] != "platform" || next[0].Endpoints[0].Metadata["zone"] != "dev-a" {
		t.Fatalf("instances leaked mutable state: %#v", next)
	}
}

func TestStaticProviderUsesExplicitServiceInstances(t *testing.T) {
	provider := NewStaticProvider(nil, WithProviderInstances([]ServiceInstance{{
		ID:      "checkout-a",
		Name:    "checkout",
		Version: "v1",
		Endpoints: []Endpoint{{
			Addr:   "127.0.0.1:50051",
			Health: HealthServing,
		}},
		Metadata: map[string]string{"owner": "platform"},
	}}))

	instances, err := provider.GetService(context.Background(), "checkout")
	if err != nil {
		t.Fatal(err)
	}
	if len(instances) != 1 || instances[0].ID != "checkout-a" || instances[0].Version != "v1" {
		t.Fatalf("unexpected instances: %#v", instances)
	}
	endpoints, err := provider.Resolve(context.Background(), "checkout")
	if err != nil {
		t.Fatal(err)
	}
	if len(endpoints) != 1 || endpoints[0].Addr != "127.0.0.1:50051" {
		t.Fatalf("unexpected endpoints from instances: %#v", endpoints)
	}
}

func TestStaticProviderResolvesSnapshotsAndWatchesService(t *testing.T) {
	provider := NewStaticProvider(map[string][]Endpoint{
		"checkout": {{
			Addr:     "127.0.0.1:50051",
			Weight:   10,
			Health:   HealthServing,
			Topology: Topology{Region: "cn", Zone: "a"},
			Metadata: map[string]string{"version": "v1"},
		}},
	}, WithProviderServices([]Service{{
		Name:      "checkout",
		Namespace: "public",
		Group:     "DEFAULT_GROUP",
		Metadata:  map[string]string{"owner": "platform"},
	}}), WithProviderMetadata(map[string]string{
		"provider": "static",
	}), WithProviderTopology(Topology{Region: "cn"}))

	var _ Provider = provider

	endpoints, err := provider.Resolve(context.Background(), "checkout")
	if err != nil {
		t.Fatal(err)
	}
	if len(endpoints) != 1 || endpoints[0].Addr != "127.0.0.1:50051" {
		t.Fatalf("unexpected endpoints: %#v", endpoints)
	}
	endpoints[0].Metadata["version"] = "mutated"

	snapshot, err := provider.Snapshot(context.Background(), "checkout")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Service != "checkout" || snapshot.Descriptor.Namespace != "public" {
		t.Fatalf("unexpected service snapshot: %#v", snapshot)
	}
	if snapshot.Descriptor.Metadata["owner"] != "platform" || snapshot.Endpoints[0].Metadata["version"] != "v1" {
		t.Fatalf("snapshot leaked mutable state: %#v", snapshot)
	}
	if !snapshot.Health.Serving || snapshot.Topology.Region != "cn" {
		t.Fatalf("unexpected health/topology: %#v", snapshot)
	}

	updates, err := provider.Watch(context.Background(), "checkout")
	if err != nil {
		t.Fatal(err)
	}
	initial, ok := <-updates
	if !ok {
		t.Fatal("expected initial discovery snapshot")
	}
	if len(initial.Endpoints) != 1 || initial.Endpoints[0].Weight != 10 {
		t.Fatalf("unexpected initial update: %#v", initial)
	}
	if _, ok := <-updates; ok {
		t.Fatal("static provider watch should close after current snapshot")
	}
}

func TestResolvePolicyFiltersVersionAndSelectorStrategies(t *testing.T) {
	endpoints := []Endpoint{
		{Addr: "a", Weight: 1, Metadata: map[string]string{"version": "v1"}},
		{Addr: "b", Weight: 5, Metadata: map[string]string{"version": "v2"}},
		{Addr: "c", Weight: 10, Metadata: map[string]string{"version": "v2"}},
	}
	policy := ResolvePolicy{Filters: []EndpointFilter{EndpointVersion("v2")}, Balancer: BalanceWeightedRoundRobin}
	selector := NewSelector(policy, WithSelectorRand(rand.New(rand.NewSource(1))))

	first, ok := selector.Select(endpoints)
	if !ok || first.Addr != "c" {
		t.Fatalf("expected highest weighted v2 endpoint first, got %#v ok=%v", first, ok)
	}
	second, ok := selector.Select(endpoints)
	if !ok || second.Addr == "a" {
		t.Fatalf("expected v2 endpoint from selector, got %#v ok=%v", second, ok)
	}
	selector.Report(first, 20*time.Millisecond, nil)
}

func TestSelectorConsistentHashUsesStableHashKey(t *testing.T) {
	endpoints := []Endpoint{
		{Addr: "10.0.0.1:50051"},
		{Addr: "10.0.0.2:50051"},
		{Addr: "10.0.0.3:50051"},
	}
	selector := NewSelector(ResolvePolicy{
		Balancer: BalanceConsistentHash,
		HashKey:  "tenant-a",
	})

	first, ok := selector.Select(endpoints)
	if !ok {
		t.Fatal("expected selected endpoint")
	}
	for i := 0; i < 20; i++ {
		next, ok := selector.Select(endpoints)
		if !ok {
			t.Fatal("expected selected endpoint")
		}
		if next.Addr != first.Addr {
			t.Fatalf("consistent hash moved from %q to %q", first.Addr, next.Addr)
		}
	}
}

func TestSelectorConsistentHashDistributesDifferentKeys(t *testing.T) {
	endpoints := []Endpoint{
		{Addr: "10.0.0.1:50051"},
		{Addr: "10.0.0.2:50051"},
		{Addr: "10.0.0.3:50051"},
		{Addr: "10.0.0.4:50051"},
	}
	selected := map[string]bool{}
	for _, key := range []string{"tenant-a", "tenant-b", "tenant-c", "tenant-d", "tenant-e", "tenant-f"} {
		endpoint, ok := NewSelector(ResolvePolicy{
			Balancer: BalanceConsistentHash,
			HashKey:  key,
		}).Select(endpoints)
		if !ok {
			t.Fatal("expected selected endpoint")
		}
		selected[endpoint.Addr] = true
	}
	if len(selected) < 2 {
		t.Fatalf("expected different hash keys to spread across endpoints, got %#v", selected)
	}
}

func TestResolvePolicySubsetSizeNarrowsEndpointsBeforeBalancing(t *testing.T) {
	endpoints := []Endpoint{
		{Addr: "10.0.0.1:50051"},
		{Addr: "10.0.0.2:50051"},
		{Addr: "10.0.0.3:50051"},
		{Addr: "10.0.0.4:50051"},
	}
	policy := ResolvePolicy{
		Balancer:   BalanceConsistentHash,
		HashKey:    "tenant-a",
		SubsetSize: 2,
	}

	filtered := ApplyResolvePolicy(endpoints, policy)
	if len(filtered) != 2 {
		t.Fatalf("subset size = %d, want 2: %#v", len(filtered), filtered)
	}
	first, ok := NewSelector(policy).Select(endpoints)
	if !ok {
		t.Fatal("expected selected endpoint")
	}
	for _, endpoint := range filtered {
		if endpoint.Addr == first.Addr {
			return
		}
	}
	t.Fatalf("selected endpoint %q outside stable subset %#v", first.Addr, filtered)
}

func TestWatchWithPolicyCanReturnStaleSnapshotWhenWatchFails(t *testing.T) {
	provider := failingWatchProvider{
		StaticProvider: NewStaticProvider(map[string][]Endpoint{
			"checkout": {{Addr: "127.0.0.1:50051", Health: HealthServing}},
		}),
	}

	updates, err := WatchWithPolicy(context.Background(), provider, "checkout", ResolvePolicy{KeepStaleOnError: true})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, ok := <-updates
	if !ok || len(snapshot.Endpoints) != 1 {
		t.Fatalf("expected stale snapshot, got %#v ok=%v", snapshot, ok)
	}
}

type failingWatchProvider struct {
	*StaticProvider
}

func (p failingWatchProvider) Watch(context.Context, string) (<-chan Snapshot, error) {
	return nil, errors.New("watch unavailable")
}

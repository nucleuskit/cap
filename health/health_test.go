package health

import (
	"context"
	"errors"
	"testing"
)

func TestAggregateReportsReadyWhenAllCapabilitiesReady(t *testing.T) {
	report, err := Aggregate(context.Background(),
		ReporterFunc(func(context.Context) (Report, error) {
			return Report{
				Capability: "sql",
				Status:     StatusReady,
				Message:    "ok",
				Metadata:   map[string]string{"role": "primary"},
			}, nil
		}),
		StaticReport(Report{Capability: "redis", Status: StatusReady}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Ready || report.Status != StatusReady {
		t.Fatalf("expected ready aggregate, got %#v", report)
	}
	if len(report.Reports) != 2 {
		t.Fatalf("expected two reports, got %#v", report.Reports)
	}

	report.Reports[0].Metadata["role"] = "mutated"
	next, err := Aggregate(context.Background(), StaticReport(Report{
		Capability: "sql",
		Status:     StatusReady,
		Metadata:   map[string]string{"role": "primary"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if next.Reports[0].Metadata["role"] != "primary" {
		t.Fatalf("metadata was not cloned: %#v", next.Reports[0].Metadata)
	}
}

func TestAggregateMarksDegradedAndNotReady(t *testing.T) {
	report, err := Aggregate(context.Background(),
		StaticReport(Report{Capability: "sql", Status: StatusReady}),
		StaticReport(Report{Capability: "redis", Status: StatusDegraded, Message: "ping timeout"}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if report.Ready || report.Status != StatusDegraded {
		t.Fatalf("expected degraded not-ready aggregate, got %#v", report)
	}
	if report.Reports[1].Message != "ping timeout" {
		t.Fatalf("expected nested message, got %#v", report.Reports[1])
	}
}

func TestAggregateIncludesReporterErrors(t *testing.T) {
	report, err := Aggregate(context.Background(),
		StaticReport(Report{Capability: "sql", Status: StatusReady}),
		ReporterFunc(func(context.Context) (Report, error) {
			return Report{Capability: "redis"}, errors.New("dial refused")
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if report.Ready || report.Status != StatusDown {
		t.Fatalf("expected down aggregate, got %#v", report)
	}
	if report.Reports[1].Status != StatusDown || report.Reports[1].Message != "dial refused" {
		t.Fatalf("expected error report, got %#v", report.Reports[1])
	}
}

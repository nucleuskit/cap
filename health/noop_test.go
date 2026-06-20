package health

import (
	"context"
	"testing"
)

func TestNoopReportsReady(t *testing.T) {
	report, err := NewNoop("log").ReportHealth(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if report.Capability != "log" || report.Status != StatusReady || !report.Ready() {
		t.Fatalf("unexpected noop report: %#v", report)
	}
}

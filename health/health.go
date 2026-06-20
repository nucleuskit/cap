package health

import "context"

type Status string

const (
	StatusUnknown  Status = "unknown"
	StatusReady    Status = "ready"
	StatusDegraded Status = "degraded"
	StatusDown     Status = "down"
)

type Report struct {
	Capability string            `json:"capability"`
	Status     Status            `json:"status"`
	Message    string            `json:"message,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type Readiness struct {
	Ready   bool     `json:"ready"`
	Status  Status   `json:"status"`
	Reports []Report `json:"reports,omitempty"`
}

type Reporter interface {
	ReportHealth(context.Context) (Report, error)
}

type ReporterFunc func(context.Context) (Report, error)

func (f ReporterFunc) ReportHealth(ctx context.Context) (Report, error) {
	return f(ctx)
}

func (r Report) Ready() bool {
	return r.Status == StatusReady
}

func StaticReport(report Report) Reporter {
	return ReporterFunc(func(context.Context) (Report, error) {
		return report.Clone(), nil
	})
}

func Aggregate(ctx context.Context, reporters ...Reporter) (Readiness, error) {
	if err := ctx.Err(); err != nil {
		return Readiness{}, err
	}
	readiness := Readiness{
		Ready:  true,
		Status: StatusReady,
	}
	for _, reporter := range reporters {
		if reporter == nil {
			continue
		}
		report, err := reporter.ReportHealth(ctx)
		if err != nil {
			report.Status = StatusDown
			report.Message = err.Error()
		}
		report = report.Clone()
		if report.Status == "" {
			report.Status = StatusUnknown
		}
		readiness.Reports = append(readiness.Reports, report)
		if !report.Ready() {
			readiness.Ready = false
		}
		readiness.Status = worstStatus(readiness.Status, report.Status)
	}
	return readiness, nil
}

func (r Report) Clone() Report {
	r.Metadata = cloneMetadata(r.Metadata)
	return r
}

func worstStatus(current Status, next Status) Status {
	if statusRank(next) > statusRank(current) {
		return next
	}
	return current
}

func statusRank(status Status) int {
	switch status {
	case StatusDown:
		return 3
	case StatusDegraded:
		return 2
	case StatusUnknown:
		return 1
	case StatusReady:
		return 0
	default:
		return 1
	}
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

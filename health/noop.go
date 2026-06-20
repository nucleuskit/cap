package health

import "context"

type noopReporter struct {
	capability string
}

func NewNoop(capability string) Reporter {
	return noopReporter{capability: capability}
}

func (r noopReporter) ReportHealth(context.Context) (Report, error) {
	return Report{
		Capability: r.capability,
		Status:     StatusReady,
	}, nil
}

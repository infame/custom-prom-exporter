package types

// MetricDefinition - global metrics struct
type MetricDefinition struct {
	Type    string
	Metrics []MetricDetail
}

// MetricDetail - keys-descriptions struct
type MetricDetail struct {
	Key         string
	Description string
}

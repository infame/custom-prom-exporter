package types

// MetricDefinition - глобальная структура метрик
type MetricDefinition struct {
	Type    string
	Metrics []MetricDetail
}

// MetricDetail - структура для определения ключей и дескрипшнов
type MetricDetail struct {
	Key         string
	Description string
}

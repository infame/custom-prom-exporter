package helpers

import "fmt"

func GetFormattedMetricName(metricType string, key string) string {
	return fmt.Sprintf("%s_%s", metricType, key)
}

func GetFormattedRedisKey(metricType string, key string) string {
	name := GetFormattedMetricName(metricType, key)
	return fmt.Sprintf("prometheus:%s", name)
}

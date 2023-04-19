package prometheus

import (
	"strconv"
	"strings"
)

// 指标导出
type Exporter struct {
	// metricCount 预估的指标个数， 即预估的本次获取的所有样本的指标名去重的个数
	metricCount uint64
	// sampleCount 预估每个指标名有多少个样本
	sampleCount   uint64
	MetricSamples map[string][]*Sample
}

// metricCount 预估的指标个数， 即预估的本次获取的所有样本的指标名去重的个数
// totalSampleCount 预估所有的样本个数
func NewExporter(metricCount, totalSampleCount uint64) *Exporter {
	preMetric := totalSampleCount / metricCount
	if preMetric < 1 {
		preMetric = 8
	}
	return &Exporter{metricCount: metricCount, sampleCount: preMetric, MetricSamples: make(map[string][]*Sample, metricCount)}
}

func (e *Exporter) String() string {
	var builder strings.Builder
	for _, samples := range e.MetricSamples {
		if len(samples) == 0 {
			continue
		}
		// 写入help
		builder.WriteString("# HELP")
		builder.WriteString(samples[0].MetricName)
		builder.WriteString(" ")
		builder.WriteString(samples[0].Help)
		builder.WriteString("\n")

		// 写入 type
		builder.WriteString("# TYPEs")
		builder.WriteString(samples[0].MetricName)
		builder.WriteString(" ")
		builder.WriteString(samples[0].Type)
		builder.WriteString("\n")

		for _, metric := range samples {
			// 写入指标
			builder.WriteString(metric.MetricName)
			builder.WriteString(mapToStr(metric.Labels))
			builder.WriteString(" ")
			builder.WriteString(strconv.FormatFloat(metric.Value, 'f', int(metric.ValuePrecision), 64))
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

// 添加 metric
func (e *Exporter) AddMetric(ms ...*Sample) {
	for _, m := range ms {
		if _, ok := e.MetricSamples[m.MetricName]; ok {
			// 已存在,追加
			e.MetricSamples[m.MetricName] = append(e.MetricSamples[m.MetricName], m)
		} else {
			// 不存在,先初始化,再追加
			e.MetricSamples[m.MetricName] = make([]*Sample, 0, e.sampleCount)
			e.MetricSamples[m.MetricName] = append(e.MetricSamples[m.MetricName], m)
		}
	}
}

// 合并 exporter
func (e *Exporter) Merge(es ...*Exporter) {
	for _, e := range es {
		for _, ms := range e.MetricSamples {
			e.AddMetric(ms...)
		}
	}
}

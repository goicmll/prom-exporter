package prometheus

// 指标数据类型
type metricType = string

var Gauge = "gauge"
var Counter = "counter"
var Histogram = "histogram"
var Summary = "summary"

// Sample 指标样本定义
type Sample struct {
	Help           string
	Type           metricType
	MetricName     string
	Labels         map[string]string
	Value          float64
	ValuePrecision int
}

// NewSample 创建指标实例
func NewSample(help string, mType metricType, mName string, labels map[string]string, value float64, valuePrecision int) *Sample {
	m := &Sample{
		Help:           help,
		Type:           mType,
		MetricName:     mName,
		Labels:         nil,
		Value:          value,
		ValuePrecision: valuePrecision,
	}
	m.addLabel(labels)
	return m
}

// 添加标签， 排除符合规范的标签
func (m *Sample) addLabel(labels ...map[string]string) {
	if m.Labels == nil {
		m.Labels = make(map[string]string)
	}
	for _, label := range labels {
		if label == nil {
			continue
		} else {
			for k, v := range label {
				m.Labels[k] = v
			}
		}
	}
}

// 删除标签
func (m *Sample) deleteLabel(labelNames []string) {
	if m.Labels == nil {
		m.Labels = make(map[string]string)
	}
	if len(labelNames) == 0 {
		return
	}
	for _, ln := range labelNames {
		delete(m.Labels, ln)
	}
}

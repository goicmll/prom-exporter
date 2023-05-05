package prometheus

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var promTagCache = make(map[string]*promTag, 1024)

// 指标数据类型
type metricType = string

var Gauge metricType = "gauge"
var Counter metricType = "counter"
var Histogram metricType = "histogram"
var Summary metricType = "summary"

// 指标样本定义
type Sample struct {
	Help           string
	Type           metricType
	MetricName     string
	Labels         map[string]string
	Value          float64
	ValuePrecision uint8
}

// 创建指标实力
func NewSample(help string, mType metricType, mName string, labels map[string]string, value float64, valuePrecision uint8) *Sample {
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

// 定义指标接口, 实现此接口的 struct 可以通过 tag 标记,自动解析成m etric
type Metricer interface {
	GetMetricNamePrefix() string
}

// 指标名和标签的匹配的正则表达式
var regexName = regexp.MustCompile(`^[A-Za-z0-9_-]{2,}$`)

// 标签值匹配的正则表达式
var regexLabelValueIgnore = regexp.MustCompile(`[{}"\\]+`)

// 验证指标名和标签名是否符合规范
func validateName(name string) bool {
	return regexName.MatchString(name)
}

// 验证标签值
func tidyLabelValue(value string) string {
	if regexLabelValueIgnore.MatchString(value) {
		return "ingnore"
	} else {
		return value
	}
}

type promTag struct {
	IsMetric       bool
	IsLabel        bool
	Help           string
	Type           metricType
	MetricName     string
	LabelName      string
	Both           bool
	ValuePrecision uint8
}

var mTypeMapping = map[string]metricType{
	"gauge":     Gauge,
	"counter":   Counter,
	"histogram": Histogram,
	"summary":   Summary,
}

// 解析 struct 的 tag 为 promTag
// prom: “help: some help;type: counter;metricName: request_total;labelName: host”
func parseTag(tagRaw string) *promTag {
	if cValue, ok := promTagCache[tagRaw]; ok {
		return cValue
	}
	var pt = promTag{
		Help:       "",
		Type:       Gauge,
		IsMetric:   false,
		IsLabel:    false,
		Both:       false,
		MetricName: "",
		LabelName:  "",
	}

	promTags := strings.Split(strings.TrimSpace(tagRaw), ";")
	for _, tag := range promTags {
		tag = strings.TrimSpace(tag)
		kv := strings.Split(tag, ":")
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "help":
			pt.Help = strings.TrimSpace(kv[1])
		case "type":
			// 默认为 gauge
			if value, ok := mTypeMapping[strings.TrimSpace(kv[1])]; ok {
				pt.Type = value
			}
		case "metricName":
			// 不符合规范的指标名忽略
			if s := strings.TrimSpace(kv[1]); validateName(s) {
				pt.MetricName = s
				pt.IsMetric = true
			}
		case "labelName":
			// 不符合规范的标签名忽略
			if s := strings.TrimSpace(kv[1]); validateName(s) {
				pt.LabelName = s
				pt.IsLabel = true
			}
		case "valuePrecision":
			if value, err := strconv.ParseUint(strings.TrimSpace(kv[1]), 10, 8); err != nil {
				pt.ValuePrecision = uint8(value)
			}
		}
	}
	// 没有声明 help, 忽略指标
	if pt.Help == "" {
		pt.IsMetric = false
	}
	// 同时是标签和指标
	if pt.IsMetric && pt.IsLabel {
		pt.Both = true
	}
	// tag 的解析缓存, prom 标签后的字符串为key
	promTagCache[tagRaw] = &pt
	return &pt
}

// 解析 metricer 为 metric
func ParseMetricer(metricer Metricer, externalLabels ...map[string]string) ([]*Sample, error) {
	var samples = make([]*Sample, 0, 32)
	label := make(map[string]string, 8)
	excludeLabel := make(map[string]string, 8)

	reflectType := reflect.TypeOf(metricer)
	reflectValue := reflect.ValueOf(metricer)

	// metricer 必须是一个 struct
	if metricer != nil && reflectValue.Kind() != reflect.Struct {
		return nil, PromError{"metricer 必须是一个 struct!"}
	}

	// 解析标签
	for i := 0; i < reflectType.NumField(); i++ {
		fieldName := reflectType.Field(i).Name
		fieldValue := reflectValue.FieldByName(fieldName)
		promTag := parseTag(reflectType.Field(i).Tag.Get("prom"))

		// 忽略
		if !(promTag.IsLabel || promTag.IsMetric) {
			continue
		}
		// 设置解析的标签
		if promTag.IsLabel {
			label[promTag.LabelName] = tidyLabelValue(fmt.Sprint(fieldValue))
		}

		// 设置解析额 指标
		if promTag.IsMetric {
			// float64 指标值
			if fv, err := strconv.ParseFloat(fmt.Sprint(fieldValue), 64); err == nil {
				s := NewSample(promTag.Help, promTag.Type, strings.Join([]string{metricer.GetMetricNamePrefix(), promTag.MetricName}, ""), nil, fv, promTag.ValuePrecision)
				samples = append(samples, s)
				// bool 指标
			} else if fv, err := strconv.ParseBool(fmt.Sprint(fieldValue)); err == nil {
				var fvf float64 = 0
				if fv {
					fvf = 1
				}
				s := NewSample(promTag.Help, promTag.Type, strings.Join([]string{metricer.GetMetricNamePrefix(), promTag.MetricName}, ""), nil, fvf, promTag.ValuePrecision)
				samples = append(samples, s)
			} else {
				msg := fmt.Sprintf("不可用的指标字段(%s)的值(%s) 必须是一个可float/bool的字段", fieldName, fmt.Sprint(fieldValue))
				return nil, PromError{msg}
			}
		}
		// 同时是指标和标签， 添加到待删除标签中
		if promTag.Both {
			excludeLabel[promTag.MetricName] = promTag.LabelName
		}
	}
	// 添加 metric 标签
	labels := append(externalLabels, label)
	for _, sample := range samples {
		sample.addLabel(labels...)
		// 删除排除的标签
		sample.deleteLabel([]string{excludeLabel[sample.MetricName]})
	}
	return samples, nil
}

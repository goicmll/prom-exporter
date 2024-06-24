package prometheus

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var promTagCache = make(map[string]*promTag, 1024)

// Metricer 定义指标接口, 实现此接口的 struct 可以通过 tag 标记,自动解析成metric
type Metricer interface {
	// GetMetricNamePrefix 指标名前缀
	GetMetricNamePrefix() string
	// GetMetricNameSuffix 指标名后缀缀
	GetMetricNameSuffix() string
	// GetMetricNameSeparator 指标名前后缀连接符
	GetMetricNameSeparator() string
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
		return "illegal"
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
	ValuePrecision int
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
			if value, err := strconv.ParseInt(strings.TrimSpace(kv[1]), 10, 8); err != nil {
				pt.ValuePrecision = int(value)
			} else {
				pt.ValuePrecision = 2
			}
		}
	}
	// 没有声明 help, 忽略指标
	if pt.Help == "" {
		pt.IsMetric = false
	}
	// tag 的解析缓存, prom 标签后的字符串为key
	promTagCache[tagRaw] = &pt
	return &pt
}

// Parse 解析 metricer 为 metric
func Parse(metricer Metricer, externalLabels ...map[string]string) ([]*Sample, error) {

	if metricer == nil {
		return make([]*Sample, 0), nil
	}
	var samples = make([]*Sample, 0, 32)

	label := make(map[string]string, 8)
	excludeLabel := make(map[string]string, 8)

	reflectType := reflect.TypeOf(metricer)
	reflectValue := reflect.ValueOf(metricer)

	// metricer 必须是一个 struct
	if reflectValue.Kind() != reflect.Struct {
		return nil, PromError{"metricer 必须是一个 struct!"}
	}

	// 解析标签
	for i := 0; i < reflectType.NumField(); i++ {
		fieldName := reflectType.Field(i).Name
		fieldValue := reflectValue.FieldByName(fieldName)
		pt := parseTag(reflectType.Field(i).Tag.Get("prom"))

		// 忽略
		if !(pt.IsLabel || pt.IsMetric) {
			continue
		}
		// 设置解析的标签
		if pt.IsLabel {
			label[pt.LabelName] = tidyLabelValue(fmt.Sprint(fieldValue))
		}
		// 指标字段解析成样本对象
		if pt.IsMetric {
			// 添加指标名前后缀
			metricName := strings.Join(
				[]string{
					metricer.GetMetricNamePrefix(),
					pt.MetricName,
					metricer.GetMetricNameSuffix(),
				},
				metricer.GetMetricNameSeparator(),
			)

			// 设置指标值
			// 可解析成float64的值的样本
			if fv, err := strconv.ParseFloat(fmt.Sprint(fieldValue), 64); err == nil {
				s := NewSample(pt.Help, pt.Type, metricName, nil, fv, pt.ValuePrecision)
				samples = append(samples, s)
				// 可解析成bool的值的样本
			} else if bv, err := strconv.ParseBool(fmt.Sprint(fieldValue)); err == nil {
				s := NewSample(pt.Help, pt.Type, metricName, nil, 0, pt.ValuePrecision)
				if bv {
					s.Value = 1
				}
				samples = append(samples, s)
			} else {
				msg := fmt.Sprintf("不可用的指标字段(%s)的值(%s) 必须是一个可float/bool的字段", fieldName, fmt.Sprint(fieldValue))
				return nil, PromError{msg}
			}
			// 同时是指标和标签， 添加到待删除标签中
			if pt.IsLabel {
				excludeLabel[metricName] = pt.LabelName
			}
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

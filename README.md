# prometheus
> prometheus exporter 生成，通过struct tag 标记生成指标样本

## example
```go
type Host struct {
	// 指标
	CPU float64 `prom:"help: cpu 使用率;type: gauge;metricName: cpu;valuePrecision:2"`
	// 指标
	Mem float64 `prom:"help: mem 使用率;type: gauge;metricName: mem;valuePrecision:2"`
	// 标签
	IP string `prom:"labelName: ip"`
	// 标签
	Hostname string `prom:"labelName: hostname"`
	// 是指标也是标签
	isOk bool `prom:"help: 是否健康;type: gauge;metricName: is_ok;labelName: is_ok"`
}

func main() {
	e := NewExporter(4, 4)
	sm := Host{}
	sm.CPU = 77.889
	sm.Mem = 88.999
	sm.IP = "1.1.1.1"
	sm.Hostname = "test"
	sm.isOk = true
	ss, err := Parse(sm, "opep_", nil)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	e.AddSamples(ss...)
	fmt.Println(e.String())
}

```
> 结果
> 
```text
# HELP opep_cpu cpu 使用率
# TYPE opep_cpu gauge
opep_cpu{hostname="test",is_ok="true",ip="1.1.1.1"} 77.89

# HELP opep_mem mem 使用率
# TYPE opep_mem gauge
opep_mem{hostname="test",is_ok="true",ip="1.1.1.1"} 89.00

# HELP opep_is_ok 是否健康
# TYPE opep_is_ok gauge
opep_is_ok{ip="1.1.1.1",hostname="test"} 1

```

## PS
1. 是指标的help必须有，没有会被忽略
2. 指标名必须匹配正则 ^[A-Za-z0-9_-]{2,}$
3. 标签值有以下字符：[{}"]'，值会被替换为 illegal
4. 指标值必须是可被转化为float64的或这是bool类型


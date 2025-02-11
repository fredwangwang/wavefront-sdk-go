package senders

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/wavefronthq/wavefront-sdk-go/event"
	"github.com/wavefronthq/wavefront-sdk-go/histogram"
	"github.com/wavefronthq/wavefront-sdk-go/internal"
)

// Gets a metric line in the Wavefront metrics data format:
// <metricName> <metricValue> [<timestamp>] source=<source> [pointTags]
// Example: "new-york.power.usage 42422.0 1533531013 source=localhost datacenter=dc1"
func MetricLine(name string, value float64, ts int64, source string, tags map[string]string, defaultSource string) (string, error) {
	if name == "" {
		return "", errors.New("empty metric name")
	}

	if source == "" {
		source = defaultSource
	}

	sb := internal.GetBuffer()
	defer internal.PutBuffer(sb)

	sb.WriteByte('"')
	sanitizeInternalSb(sb, name)
	sb.WriteByte('"')

	sb.WriteByte(' ')
	sb.SetBuf(strconv.AppendFloat(sb.GetBuf(), value, 'f', -1, 64))

	if ts != 0 {
		sb.WriteByte(' ')
		sb.SetBuf(strconv.AppendInt(sb.GetBuf(), ts, 10))
	}

	sb.WriteString(" source=")
	sanitizeValueSb(sb, source)

	for k, v := range tags {
		if v == "" {
			return "", errors.New("metric point tag value cannot be blank")
		}
		sb.WriteByte(' ')

		sb.WriteByte('"')
		sanitizeInternalSb(sb, k)
		sb.WriteByte('"')
		sb.WriteByte('=')
		sanitizeValueSb(sb, v)
	}
	sb.WriteByte('\n')
	return sb.String(), nil
}

// Gets a histogram line in the Wavefront histogram data format:
// {!M | !H | !D} [<timestamp>] #<count> <mean> [centroids] <histogramName> source=<source> [pointTags]
// Example: "!M 1533531013 #20 30.0 #10 5.1 request.latency source=appServer1 region=us-west"
func HistoLine(name string, centroids histogram.Centroids, hgs map[histogram.Granularity]bool, ts int64, source string, tags map[string]string, defaultSource string) (string, error) {
	if name == "" {
		return "", errors.New("empty distribution name")
	}

	if len(centroids) == 0 {
		return "", errors.New("distribution should have at least one centroid")
	}

	if len(hgs) == 0 {
		return "", errors.New("histogram granularities cannot be empty")
	}

	if source == "" {
		source = defaultSource
	}

	sb := internal.GetBuffer()
	defer internal.PutBuffer(sb)

	if ts != 0 {
		sb.WriteByte(' ')
		sb.SetBuf(strconv.AppendInt(sb.GetBuf(), ts, 10))
	}
	// Preprocess line. We know len(hgs) > 0 here.
	for _, centroid := range centroids.Compact() {
		sb.WriteString(" #")
		sb.SetBuf(strconv.AppendInt(sb.GetBuf(), int64(centroid.Count), 10))
		sb.WriteByte(' ')
		sb.SetBuf(strconv.AppendFloat(sb.GetBuf(), centroid.Value, 'f', -1, 64))
	}
	sb.WriteByte(' ')
	sb.WriteByte('"')
	sanitizeInternalSb(sb, name)
	sb.WriteByte('"')

	sb.WriteString(" source=")
	sanitizeValueSb(sb, source)

	for k, v := range tags {
		if v == "" {
			return "", errors.New("histogram tag value cannot be blank")
		}
		sb.WriteByte(' ')
		sb.WriteByte('"')
		sanitizeInternalSb(sb, k)
		sb.WriteByte('"')
		sb.WriteByte('=')
		sanitizeValueSb(sb, v)
	}
	sbBytes := sb.GetBuf()

	sbg := bytes.Buffer{}
	for hg, on := range hgs {
		if on {
			sbg.WriteString(hg.String())
			sbg.Write(sbBytes)
			sbg.WriteByte('\n')
		}
	}
	return sbg.String(), nil
}

// Gets a span line in the Wavefront span data format:
// <tracingSpanName> source=<source> [pointTags] <start_millis> <duration_milli_seconds>
// Example:
// "getAllUsers source=localhost traceId=7b3bf470-9456-11e8-9eb6-529269fb1459 spanId=0313bafe-9457-11e8-9eb6-529269fb1459
//    parent=2f64e538-9457-11e8-9eb6-529269fb1459 application=Wavefront http.method=GET 1533531013 343500"
func SpanLine(name string, startMillis, durationMillis int64, source, traceId, spanId string, parents, followsFrom []string, tags []SpanTag, spanLogs []SpanLog, defaultSource string) (string, error) {
	if name == "" {
		return "", errors.New("empty span name")
	}

	if source == "" {
		source = defaultSource
	}

	if !isUUIDFormat(traceId) {
		return "", errors.New("traceId is not in UUID format")
	}
	if !isUUIDFormat(spanId) {
		return "", errors.New("spanId is not in UUID format")
	}

	sb := internal.GetBuffer()
	defer internal.PutBuffer(sb)

	sanitizeValueSb(sb, name)
	sb.WriteString(" source=")
	sanitizeValueSb(sb, source)
	sb.WriteString(" traceId=")
	sb.WriteString(traceId)
	sb.WriteString(" spanId=")
	sb.WriteString(spanId)

	for _, parent := range parents {
		sb.WriteString(" parent=")
		sb.WriteString(parent)
	}

	for _, item := range followsFrom {
		sb.WriteString(" followsFrom=")
		sb.WriteString(item)
	}

	if len(spanLogs) > 0 {
		sb.WriteByte(' ')
		sb.WriteByte('"')
		sb.WriteString("_spanLogs")
		sb.WriteByte('"')
		sb.WriteByte('=')
		sb.WriteByte('"')
		sb.WriteString("true")
		sb.WriteByte('"')
	}

	for _, tag := range tags {
		if tag.Key == "" || tag.Value == "" {
			return "", errors.New("span tag key/value cannot be blank")
		}
		sb.WriteByte(' ')
		sb.WriteByte('"')
		sanitizeInternalSb(sb, tag.Key)
		sb.WriteByte('"')
		sb.WriteByte('=')
		sanitizeValueSb(sb, tag.Value)
	}
	sb.WriteByte(' ')
	sb.SetBuf(strconv.AppendInt(sb.GetBuf(), startMillis, 10))
	sb.WriteByte(' ')
	sb.SetBuf(strconv.AppendInt(sb.GetBuf(), durationMillis, 10))
	sb.WriteByte('\n')

	return sb.String(), nil
}

func SpanLogJSON(traceId, spanId string, spanLogs []SpanLog) (string, error) {
	l := SpanLogs{
		TraceId: traceId,
		SpanId:  spanId,
		Logs:    spanLogs,
	}
	out, err := json.Marshal(l)
	if err != nil {
		return "", err
	}
	return string(out[:]) + "\n", nil
}

// EventLine encode the event to a wf proxy format
// set endMillis to 0 for a 'Instantaneous' event
func EventLine(name string, startMillis, endMillis int64, source string, tags map[string]string, setters ...event.Option) (string, error) {
	sb := internal.GetBuffer()
	defer internal.PutBuffer(sb)

	annotations := map[string]string{}
	l := map[string]interface{}{
		"annotations": annotations,
	}
	for _, set := range setters {
		set(l)
	}

	sb.WriteString("@Event")

	startMillis, endMillis = adjustStartEndTime(startMillis, endMillis)

	sb.WriteByte(' ')
	sb.SetBuf(strconv.AppendInt(sb.GetBuf(), startMillis, 10))
	sb.WriteByte(' ')
	sb.SetBuf(strconv.AppendInt(sb.GetBuf(), endMillis, 10))
	sb.WriteByte(' ')
	sb.WriteString(strconv.Quote(name))

	for k, v := range annotations {
		sb.WriteByte(' ')
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(strconv.Quote(v))
	}

	if len(source) > 0 {
		sb.WriteString(" host=")
		sb.WriteString(strconv.Quote(source))
	}

	for k, v := range tags {
		sb.WriteString(" tag=")
		sb.WriteString(strconv.Quote(fmt.Sprintf("%v: %v", k, v)))
	}

	sb.WriteByte('\n')
	return sb.String(), nil
}

// EventLine encode the event to a wf API format
// set endMillis to 0 for a 'Instantaneous' event
func EventLineJSON(name string, startMillis, endMillis int64, source string, tags map[string]string, setters ...event.Option) (string, error) {
	annotations := map[string]string{}
	l := map[string]interface{}{
		"name":        name,
		"annotations": annotations,
	}

	for _, set := range setters {
		set(l)
	}

	startMillis, endMillis = adjustStartEndTime(startMillis, endMillis)

	l["startTime"] = startMillis
	l["endTime"] = endMillis

	if len(tags) > 0 {
		var tagList []string
		for k, v := range tags {
			tagList = append(tagList, fmt.Sprintf("%v: %v", k, v))
		}
		l["tags"] = tagList
	}

	if len(source) > 0 {
		l["hosts"] = []string{source}
	}

	jsonData, err := json.Marshal(l)
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

func adjustStartEndTime(startMillis, endMillis int64) (int64, int64) {
	// secs to millis
	if startMillis < 999999999999 {
		startMillis = startMillis * 1000
	}

	if endMillis <= 999999999999 {
		endMillis = endMillis * 1000
	}

	if endMillis == 0 {
		endMillis = startMillis + 1
	}
	return startMillis, endMillis
}

func isUUIDFormat(str string) bool {
	l := len(str)
	if l != 36 {
		return false
	}
	for i := 0; i < l; i++ {
		c := str[i]
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else if !(('0' <= c && c <= '9') || ('a' <= c && c <= 'f') || ('A' <= c && c <= 'F')) {
			return false
		}
	}
	return true
}

//Sanitize string of metric name, source and key of tags according to the rule of Wavefront proxy.
func sanitizeInternal(str string) string {
	sb := internal.GetBuffer()
	defer internal.PutBuffer(sb)

	// first character can be \u2206 (∆ - INCREMENT) or \u0394 (Δ - GREEK CAPITAL LETTER DELTA)
	// or ~ tilda character for internal metrics
	skipHead := 0
	if strings.HasPrefix(str, internal.DeltaPrefix) {
		sb.WriteString(internal.DeltaPrefix)
		skipHead = 3
	}
	if strings.HasPrefix(str, internal.AltDeltaPrefix) {
		sb.WriteString(internal.AltDeltaPrefix)
		skipHead = 2
	}
	// The first char after \u2206 (∆ - INCREMENT) or \u0394 (Δ - GREEK CAPITAL LETTER) (if there is any)
	// can be ~ tilda character
	if str[skipHead] == '~' {
		sb.WriteByte('~')
		skipHead += 1
	}

	for i := skipHead; i < len(str); i++ {
		cur := str[i]
		strCur := string(cur)
		isLegal := true

		if !(',' <= cur && cur <= '9') && !('A' <= cur && cur <= 'Z') && !('a' <= cur && cur <= 'z') && cur != '_' {
			isLegal = false
		}
		if isLegal {
			sb.WriteString(strCur)
		} else {
			sb.WriteByte('-')
		}
	}
	return sb.String()
}

//Sanitize string of metric name, source and key of tags according to the rule of Wavefront proxy.
func sanitizeInternalSb(sb *internal.StringBuilder, str string) {
	// first character can be \u2206 (∆ - INCREMENT) or \u0394 (Δ - GREEK CAPITAL LETTER DELTA)
	// or ~ tilda character for internal metrics
	skipHead := 0
	if strings.HasPrefix(str, internal.DeltaPrefix) {
		sb.WriteString(internal.DeltaPrefix)
		skipHead = 3
	}
	if strings.HasPrefix(str, internal.AltDeltaPrefix) {
		sb.WriteString(internal.AltDeltaPrefix)
		skipHead = 2
	}
	// The first char after \u2206 (∆ - INCREMENT) or \u0394 (Δ - GREEK CAPITAL LETTER) (if there is any)
	// can be ~ tilda character
	if str[skipHead] == '~' {
		sb.WriteByte('~')
		skipHead += 1
	}

	for i := skipHead; i < len(str); i++ {
		cur := str[i]
		isLegal := true

		if !(',' <= cur && cur <= '9') && !('A' <= cur && cur <= 'Z') && !('a' <= cur && cur <= 'z') && cur != '_' {
			isLegal = false
		}
		if isLegal {
			sb.WriteByte(cur)
		} else {
			sb.WriteByte('-')
		}
	}
}

//Sanitize string of tags value, etc.
func sanitizeValue(str string) string {
	res := strings.TrimSpace(str)
	if strings.Contains(str, "\"") {
		res = strings.ReplaceAll(res, `"`, `\"`)
	}

	return "\"" + strings.ReplaceAll(res, "\n", "\\n") + "\""
}

//Sanitize string of tags value, etc.
func sanitizeValueSb(sb *internal.StringBuilder, str string) {
	res := strings.TrimSpace(str)
	if strings.Contains(str, "\"") {
		res = strings.ReplaceAll(res, `"`, `\"`)
	}
	sb.WriteByte('"')
	sb.WriteString(strings.ReplaceAll(res, "\n", "\\n"))
	sb.WriteByte('"')
}

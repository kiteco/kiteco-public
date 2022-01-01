// +build windows

package performance

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"time"

	"github.com/StackExchange/wmi"
)

func fanSpeedsImpl() ([]FanSpeedStat, error) {
	type Win32Fan struct {
		ActiveCooling bool
		Availability  uint16
		Caption       string
		Description   string
		DesiredSpeed  uint64
		DeviceID      string
		Name          string
		Status        string
		VariableSpeed bool
	}

	var dst []Win32Fan
	q := createQueryForClass(&dst, "", "Win32_Fan")
	ctxTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- wmi.Query(q, &dst, nil, "root/cimv2")
	}()

	select {
	case <-ctxTimeout.Done():
		return nil, ctxTimeout.Err()
	case err := <-errChan:
		return nil, err
	}
	result := make([]FanSpeedStat, 0, len(dst))
	for _, s := range dst {
		result = append(result, FanSpeedStat{
			SensorKey: s.Name,
			Speed:     float64(s.DesiredSpeed),
		})
	}
	return result, nil
}

// createQueryForClass is a copy of wmi.CreateQuery with using className instead of src.type() to avoid having
// to name the struct with the exact WMI class name (issue with golint because of the _ in the names)
func createQueryForClass(src interface{}, where string, className string) string {
	var b bytes.Buffer
	b.WriteString("SELECT ")
	s := reflect.Indirect(reflect.ValueOf(src))
	t := s.Type()
	if s.Kind() == reflect.Slice {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return ""
	}
	var fields []string
	for i := 0; i < t.NumField(); i++ {
		fields = append(fields, t.Field(i).Name)
	}
	b.WriteString(strings.Join(fields, ", "))
	b.WriteString(" FROM ")
	b.WriteString(className)
	b.WriteString(" " + where)
	return b.String()
}

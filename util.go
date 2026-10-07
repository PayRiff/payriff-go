package payriff

import (
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const MaxPageSize = 20

type body map[string]any

func (b body) set(key string, value any) body {
	v := reflect.ValueOf(value)
	if !v.IsValid() || v.IsZero() || (v.Kind() == reflect.Map && v.Len() == 0) {
		return b
	}
	b[key] = value
	return b
}

func segment(value, name string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", invalid("%s must not be blank", name)
	}
	return url.PathEscape(value), nil
}

func required(value, name string) error {
	if strings.TrimSpace(value) == "" {
		return invalid("%s is required", name)
	}
	return nil
}

func positive(value float64, name string) error {
	if value <= 0 {
		return invalid("%s is required", name)
	}
	return nil
}

func orDefault[T ~string](value, fallback T) T {
	if value == "" {
		return fallback
	}
	return value
}

func filterDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("02.01.2006")
}

func pageQuery(page, size int) ([][2]string, error) {
	if size == 0 {
		size = 10
	}
	if page < 0 {
		return nil, invalid("page must be >= 0")
	}
	if size < 1 || size > MaxPageSize {
		return nil, invalid("size must be 1-%d", MaxPageSize)
	}
	return [][2]string{{"page", strconv.Itoa(page)}, {"offset", strconv.Itoa(size)}}, nil
}

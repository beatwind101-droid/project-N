package util

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ToString 将值转换为字符串
func ToString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	default:
		jsonBytes, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(jsonBytes)
	}
}

// ToInt 将值转换为 int
func ToInt(v interface{}) int {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case string:
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return int(f)
		}
		return 0
	case bool:
		if val {
			return 1
		}
		return 0
	case json.Number:
		if i, err := val.Int64(); err == nil {
			return int(i)
		}
		if f, err := val.Float64(); err == nil {
			return int(f)
		}
		return 0
	default:
		return 0
	}
}

// ToBool 将值转换为 bool
func ToBool(v interface{}) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		lower := strings.ToLower(val)
		return lower == "true" || lower == "1" || lower == "yes" || lower == "on"
	case int:
		return val != 0
	case int64:
		return val != 0
	case float64:
		return val != 0
	default:
		return false
	}
}

// ToStringSlice 将值转换为 []string
func ToStringSlice(v interface{}) []string {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case []string:
		return val
	case []interface{}:
		result := make([]string, len(val))
		for i, item := range val {
			result[i] = ToString(item)
		}
		return result
	case string:
		if strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]") {
			var result []string
			if err := json.Unmarshal([]byte(val), &result); err == nil {
				return result
			}
		}
		return strings.Split(val, ",")
	default:
		return nil
	}
}

// ToStringMap 将值转换为 map[string]string
func ToStringMap(v interface{}) map[string]string {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case map[string]string:
		return val
	case map[string]interface{}:
		result := make(map[string]string)
		for k, v := range val {
			result[k] = ToString(v)
		}
		return result
	case string:
		if strings.HasPrefix(val, "{") && strings.HasSuffix(val, "}") {
			var result map[string]string
			if err := json.Unmarshal([]byte(val), &result); err == nil {
				return result
			}
		}
		return nil
	default:
		return nil
	}
}

// GetConfigValue 从配置中获取值，支持默认值
func GetConfigValue[T any](config map[string]interface{}, key string, defaultValue T, converter func(interface{}) T) T {
	if v, ok := config[key]; ok {
		return converter(v)
	}
	return defaultValue
}

package web

import (
	"strconv"
	"strings"
	"time"
)

// Time 序列化为与 JavaScript Date.toJSON() 一致的 UTC 毫秒格式，
// 使前端拿到的时间字符串与旧后端逐字节相同。
type Time time.Time

func (t Time) MarshalJSON() ([]byte, error) {
	s := time.Time(t).UTC().Format("2006-01-02T15:04:05.000Z")
	return []byte(strconv.Quote(s)), nil
}

// UnmarshalJSON 解析 UTC 毫秒格式，供详情缓存往返使用。
func (t *Time) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	if s == "" || s == "null" {
		*t = Time(time.Time{})
		return nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return err
	}
	*t = Time(parsed)
	return nil
}

// TimeOf 把数据库时间转换为响应时间。
func TimeOf(t time.Time) Time { return Time(t) }

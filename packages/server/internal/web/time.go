package web

import (
	"strconv"
	"strings"
	"time"
)

type Time time.Time

func (t Time) MarshalJSON() ([]byte, error) {
	s := time.Time(t).UTC().Format("2006-01-02T15:04:05.000Z")
	return []byte(strconv.Quote(s)), nil
}

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

func TimeOf(t time.Time) Time { return Time(t) }

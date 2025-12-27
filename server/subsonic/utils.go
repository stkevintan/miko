package subsonic

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

func getQueryInt[T Integer](c *gin.Context, key string, defaultValue ...T) (T, error) {
	valStr := c.Query(key)
	if valStr == "" {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		var zero T
		return zero, fmt.Errorf("missing required parameter: %s", key)
	}

	var zero T
	var res T
	var err error

	switch any(zero).(type) {
	case uint, uint8, uint16, uint32, uint64:
		var val uint64
		val, err = strconv.ParseUint(valStr, 10, 64)
		res = T(val)
	default:
		var val int64
		val, err = strconv.ParseInt(valStr, 10, 64)
		res = T(val)
	}

	if err != nil {
		return zero, fmt.Errorf("invalid value for parameter '%s': %s", key, valStr)
	}
	return res, nil
}

func getQueryIntOrDefault[T Integer](c *gin.Context, key string, defaultValue T, err *error) T {
	if *err != nil {
		var zero T
		return zero
	}
	var val T
	val, *err = getQueryInt(c, key, defaultValue)
	return val
}

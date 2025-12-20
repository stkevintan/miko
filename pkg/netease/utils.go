package netease

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/chaunsin/netease-cloud-music/api/types"
	nmTypes "github.com/chaunsin/netease-cloud-music/api/types"
)

// ValidateQualityLevel validates and converts quality level string to types.Level
func ValidateQualityLevel(level string) (nmTypes.Level, error) {
	if level == "" {
		return types.LevelHires, nil // default to hires
	}

	// Handle numeric levels
	if lv, err := strconv.ParseInt(string(level), 10, 64); err == nil {
		switch lv {
		case 128:
			return types.LevelStandard, nil
		case 192:
			return types.LevelHigher, nil
		case 320:
			return types.LevelExhigh, nil
		default:
			return "", fmt.Errorf("%v level is not supported", lv)
		}
	}

	// Handle string levels
	switch types.Level(level) {
	case types.LevelStandard,
		types.LevelHigher,
		types.LevelExhigh,
		types.LevelLossless,
		types.LevelHires:
		return types.Level(level), nil
	default:
		// Handle uppercase aliases
		switch strings.ToUpper(level) {
		case "HQ":
			return types.LevelExhigh, nil
		case "SQ":
			return types.LevelLossless, nil
		case "HR":
			return types.LevelHires, nil
		default:
			return "", fmt.Errorf("[%s] quality is not supported", level)
		}
	}
}

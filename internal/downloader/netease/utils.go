package netease

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/chaunsin/netease-cloud-music/api/types"
	nmTypes "github.com/chaunsin/netease-cloud-music/api/types"
)

var (
	urlPattern = "/(song|artist|album|playlist)\\?id=(\\d+)"
	reg        = regexp.MustCompile(urlPattern)
)

// ParseURI parses a NetEase music URI and returns the type and ID
func ParseURI(source string) (string, int64, error) {
	// 歌曲id
	id, err := strconv.ParseInt(source, 10, 64)
	if err == nil {
		return "song", id, nil
	}

	if !strings.Contains(source, "music.163.com") {
		return "", 0, fmt.Errorf("could not parse the url: %s", source)
	}

	matched, ok := reg.FindStringSubmatch(source), reg.MatchString(source)
	if !ok || len(matched) < 3 {
		return "", 0, fmt.Errorf("could not parse the url: %s", source)
	}

	id, err = strconv.ParseInt(matched[2], 10, 64)
	if err != nil {
		return "", 0, err
	}
	return matched[1], id, nil
}

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

// formatArtists formats a list of artists into a comma-separated string
func formatArtists(artists []types.Artist) string {
	if len(artists) == 0 {
		return "Unknown Artist"
	}

	var names []string
	for _, artist := range artists {
		names = append(names, artist.Name)
	}
	return strings.Join(names, ", ")
}

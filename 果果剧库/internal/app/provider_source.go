package app

import (
	"net/http"
	"strings"
)

const (
	sourceHongguo  = "hongguo"
	sourceHuangguo = "huangguo"
	sourceHuangdou = "huangdou"
)

func knownLibrarySource(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case sourceHongguo, "hongguoduanju.com", "www.hongguoduanju.com":
		return sourceHongguo
	case sourceHuangguo, "huangguo.video", "www.huangguo.video":
		return sourceHuangguo
	case sourceHuangdou, "hddj.tv", "www.hddj.tv":
		return sourceHuangdou
	default:
		return ""
	}
}

func canonicalProviderSource(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case sourceHongguo, "hongguoduanju.com", "www.hongguoduanju.com":
		return sourceHongguo
	default:
		return ""
	}
}

// viewableProviderSource keeps a safe, normalized label for cached providers.
// Only canonicalProviderSource is used for network access; this helper is for
// displaying previously stored resources without silently discarding them.
func viewableProviderSource(source string) string {
	if known := knownLibrarySource(source); known != "" {
		return known
	}
	if canonical := canonicalProviderSource(source); canonical != "" {
		return canonical
	}
	source = strings.ToLower(strings.TrimSpace(source))
	if source == "" || len(source) > 64 {
		return ""
	}
	for _, char := range source {
		if char >= 'a' && char <= 'z' || char >= '0' && char <= '9' || char == '-' || char == '_' || char == '.' {
			continue
		}
		return ""
	}
	return source
}

func splitProviderDramaID(identifier string) (source, sourceID string, ok bool) {
	identifier = strings.TrimSpace(identifier)
	source, sourceID, prefixed := strings.Cut(identifier, ":")
	if prefixed {
		if canonicalProviderSource(source) != sourceHongguo {
			return "", "", false
		}
	} else {
		sourceID = identifier
	}
	sourceID = strings.TrimSpace(strings.TrimPrefix(sourceID, "hg-series-v1:"))
	if !hongguoNumericID.MatchString(sourceID) {
		return "", "", false
	}
	return sourceHongguo, sourceID, true
}

func normalizeHongguoDrama(drama Drama) (Drama, bool) {
	if drama.Source != "" && canonicalProviderSource(drama.Source) != sourceHongguo {
		return Drama{}, false
	}
	_, sourceID, valid := splitProviderDramaID(drama.ID)
	if !valid {
		return Drama{}, false
	}
	drama.Source = sourceHongguo
	drama.SourceID = sourceID
	drama.ID = providerDramaID(sourceHongguo, sourceID)
	return drama, true
}

func onlyHongguoDramas(dramas []Drama) []Drama {
	filtered := make([]Drama, 0, len(dramas))
	seen := make(map[string]bool, len(dramas))
	for _, drama := range dramas {
		if normalized, valid := normalizeHongguoDrama(drama); valid && !seen[normalized.ID] {
			seen[normalized.ID] = true
			filtered = append(filtered, normalized)
		}
	}
	return filtered
}

func normalizeViewableDrama(drama Drama) (Drama, bool) {
	if normalized, valid := normalizeHongguoDrama(drama); valid {
		return normalized, true
	}
	drama.ID = strings.TrimSpace(drama.ID)
	if drama.ID == "" || len(drama.ID) > 512 {
		return Drama{}, false
	}
	prefix, sourceID, prefixed := strings.Cut(drama.ID, ":")
	idSource := ""
	if prefixed {
		idSource = viewableProviderSource(prefix)
		if idSource == "" || strings.TrimSpace(sourceID) == "" {
			return Drama{}, false
		}
	}
	source := viewableProviderSource(drama.Source)
	if source == "" {
		source = idSource
	}
	if source == "" || idSource != "" && source != idSource {
		return Drama{}, false
	}
	drama.Source = source
	if drama.SourceID == "" && prefixed {
		drama.SourceID = strings.TrimSpace(sourceID)
	}
	return drama, true
}

func viewableDramas(dramas []Drama) []Drama {
	filtered := make([]Drama, 0, len(dramas))
	seen := make(map[string]bool, len(dramas))
	for _, drama := range dramas {
		if normalized, valid := normalizeViewableDrama(drama); valid && !seen[normalized.ID] {
			seen[normalized.ID] = true
			filtered = append(filtered, normalized)
		}
	}
	return filtered
}

func isHongguoTask(task Task) bool {
	if task.Chapter.Source != "" && canonicalProviderSource(task.Chapter.Source) != sourceHongguo {
		return false
	}
	_, _, valid := splitProviderDramaID(task.DramaID)
	return valid
}

func readHongguoIDsRequest(writer http.ResponseWriter, request *http.Request) ([]string, bool) {
	identifiers, valid := readIDsRequest(writer, request)
	if !valid {
		return nil, false
	}
	for index, identifier := range identifiers {
		_, sourceID, supported := splitProviderDramaID(identifier)
		if !supported {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "此版本仅支持红果剧集 ID"})
			return nil, false
		}
		identifiers[index] = providerDramaID(sourceHongguo, sourceID)
	}
	return identifiers, true
}

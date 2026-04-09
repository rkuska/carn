package claude

import "strconv"

func rawJSONStringSet(values map[string]struct{}) map[string]struct{} {
	raw := make(map[string]struct{}, len(values))
	for value := range values {
		raw[strconv.Quote(value)] = struct{}{}
	}
	return raw
}

func rawJSONStringValueMap(values map[string]string) map[string]string {
	raw := make(map[string]string, len(values))
	for _, value := range values {
		raw[strconv.Quote(value)] = value
	}
	return raw
}

func hasRawJSONString(raw []byte, known map[string]struct{}) bool {
	if len(raw) == 0 {
		return false
	}
	_, ok := known[bytesToStringView(raw)]
	return ok
}

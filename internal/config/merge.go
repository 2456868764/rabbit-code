package config

// DeepMerge merges src into dst (dst modified). Nested JSON objects merge recursively;
// scalars and non-map values from src replace dst keys.
func DeepMerge(dst, src map[string]interface{}) {
	if dst == nil || src == nil {
		return
	}
	for k, v := range src {
		if v == nil {
			delete(dst, k)
			continue
		}
		srcMap, srcOk := asStringMap(v)
		if srcOk {
			if dm, ok := asStringMap(dst[k]); ok {
				DeepMerge(dm, srcMap)
				dst[k] = dm
				continue
			}
			dst[k] = cloneMap(srcMap)
			continue
		}
		dst[k] = v
	}
}

func asStringMap(v interface{}) (map[string]interface{}, bool) {
	m, ok := v.(map[string]interface{})
	return m, ok
}

func cloneMap(m map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	DeepMerge(out, m)
	return out
}

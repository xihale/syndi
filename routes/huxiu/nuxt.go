package routes

import (
	"encoding/json"
	"maps"
	"regexp"
	"strings"
)

// nuxtDoc wraps a parsed Nuxt 3 __NUXT_DATA__ payload (devalue format),
// where object/array values that are plain numbers are references to other
// entries in the top-level array.
type nuxtDoc struct {
	arr []any
}

var nuxtScriptRe = regexp.MustCompile(`(?s)<script[^>]*id="__NUXT_DATA__"[^>]*>(.*?)</script>`)

// parseNuxtData extracts and parses __NUXT_DATA__ from an HTML page.
// Returns nil when the page has no such payload.
func parseNuxtData(pageHTML string) *nuxtDoc {
	m := nuxtScriptRe.FindStringSubmatch(pageHTML)
	if m == nil {
		return nil
	}
	var arr []any
	if err := json.Unmarshal([]byte(m[1]), &arr); err != nil {
		return nil
	}
	return &nuxtDoc{arr: arr}
}

// findObjectWithKey returns the resolved object of the first top-level entry
// whose key (e.g. "articleDetail-4885592") starts with prefix.
func (n *nuxtDoc) findObjectWithKey(prefix string) map[string]any {
	if n == nil {
		return nil
	}
	for _, it := range n.arr {
		obj, ok := it.(map[string]any)
		if !ok {
			continue
		}
		for k, v := range obj {
			if strings.HasPrefix(k, prefix) {
				if idx, ok := v.(float64); ok {
					if r, ok := n.resolve(int(idx), map[int]struct{}{}).(map[string]any); ok {
						return r
					}
				}
			}
		}
	}
	return nil
}

// resolve dereferences Nuxt/devalue numeric references starting at index,
// mirroring the reference implementation: every number inside an object is
// treated as an index into the array; ShallowReactive/Reactive wrappers are
// unwrapped; visited sets are cloned per value like upstream.
func (n *nuxtDoc) resolve(idx int, visited map[int]struct{}) any {
	if _, ok := visited[idx]; ok {
		return n.arr[idx]
	}
	visited[idx] = struct{}{}
	item := n.arr[idx]
	switch v := item.(type) {
	case []any:
		if len(v) >= 1 {
			if tag, ok := v[0].(string); ok {
				switch tag {
				case "ShallowReactive", "Reactive":
					if len(v) == 2 {
						if r, ok := v[1].(float64); ok {
							return n.resolve(int(r), visited)
						}
					}
				case "Set":
					return []any{}
				}
			}
		}
		out := make([]any, 0, len(v))
		for _, val := range v {
			if num, ok := val.(float64); ok && int(num) < len(n.arr) {
				out = append(out, n.resolve(int(num), maps.Clone(visited)))
			} else {
				out = append(out, val)
			}
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, val := range v {
			if num, ok := val.(float64); ok && int(num) < len(n.arr) {
				out[k] = n.resolve(int(num), maps.Clone(visited))
			} else {
				out[k] = val
			}
		}
		return out
	default:
		return item
	}
}

// nested resolves one more indirection inside a resolved object.
func nested(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	if r, ok := m[key].(map[string]any); ok {
		return r
	}
	return nil
}

// anyToString coerces a resolved JSON value to string.
func anyToString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		b, _ := json.Marshal(t)
		return strings.Trim(string(b), `".0`)
	default:
		return ""
	}
}

// anyToTimeUnix coerces a resolved JSON value to unix seconds.
func anyToTimeUnix(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case string:
		var f flexInt64
		if err := json.Unmarshal([]byte(`"`+t+`"`), &f); err == nil {
			return f.Int()
		}
	}
	return 0
}

// articleDetailFromNuxt locates the articleDetail payload inside a Huxiu
// article page's NUXT data. Structure:
// [["ShallowReactive",1],{data:2,...},...,{"articleDetail-<id>":4},...,{articleDetail:5}]
func articleDetailFromNuxt(pageHTML string) map[string]any {
	doc := parseNuxtData(pageHTML)
	obj := doc.findObjectWithKey("articleDetail-")
	return nested(obj, "articleDetail")
}

// nuxtString reads a string field from a resolved object.
func nuxtString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if s := anyToString(m[k]); s != "" {
			return s
		}
	}
	return ""
}

// nuxtTags extracts tag names from tags_info / relation_info.channel.
func nuxtTags(m map[string]any) []string {
	seen := map[string]bool{}
	var out []string
	appendTag := func(name string) {
		if name != "" && !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	if list, ok := m["tags_info"].([]any); ok {
		for _, t := range list {
			if tm, ok := t.(map[string]any); ok {
				appendTag(anyToString(tm["tag_name"]))
			}
		}
	}
	if rel, ok := m["relation_info"].(map[string]any); ok {
		if chans, ok := rel["channel"].([]any); ok {
			for _, ch := range chans {
				if cm, ok := ch.(map[string]any); ok {
					appendTag(anyToString(cm["name"]))
				}
			}
		}
	}
	return out
}

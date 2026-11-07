package config

import (
	"fmt"
	"reflect"
)

// NonReloadableKeys lists config keys that require a daemon restart to take effect.
var NonReloadableKeys = []string{
	"session.compositor",
	"paths.config_dir",
	"paths.data_dir",
	"paths.state_dir",
	"paths.cache_dir",
}

// IsNonReloadable reports whether key is non-reloadable.
func IsNonReloadable(key string) bool {
	for _, k := range NonReloadableKeys {
		if k == key {
			return true
		}
	}
	return false
}

// ConfigDiff describes changed keys between two config snapshots.
type ConfigDiff struct {
	Changed         []string
	NonReloadable   []string
	ReloadableCount int
}

// DiffDesktop computes the diff between old and new desktop configs.
func DiffDesktop(old, new DesktopConfig) ConfigDiff {
	d := ConfigDiff{}
	walkStruct("", reflect.ValueOf(old), reflect.ValueOf(new), &d)
	for _, k := range d.Changed {
		if IsNonReloadable(k) {
			d.NonReloadable = append(d.NonReloadable, k)
		} else {
			d.ReloadableCount++
		}
	}
	return d
}

func walkStruct(prefix string, a, b reflect.Value, d *ConfigDiff) {
	if a.Type() != b.Type() {
		return
	}
	t := a.Type()
	if t.Kind() != reflect.Struct {
		if !reflect.DeepEqual(a.Interface(), b.Interface()) {
			if prefix != "" {
				d.Changed = append(d.Changed, prefix)
			}
		}
		return
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		yamlTag := f.Tag.Get("yaml")
		if yamlTag == "" || yamlTag == "-" {
			continue
		}
		name := yamlTag
		if idx := indexByte(name, ','); idx >= 0 {
			name = name[:idx]
		}
		key := name
		if prefix != "" {
			key = prefix + "." + name
		}
		walkStruct(key, a.Field(i), b.Field(i), d)
	}
}

func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// ReloadResult summarizes a reload attempt.
type ReloadResult struct {
	Diff   ConfigDiff
	Applied bool
	Reason  string
}

// ValidateReload checks diff and reports non-reloadable keys.
func ValidateReload(old, new DesktopConfig) ReloadResult {
	diff := DiffDesktop(old, new)
	if len(diff.NonReloadable) > 0 {
		return ReloadResult{Diff: diff, Applied: false, Reason: fmt.Sprintf("non-reloadable keys changed: %v", diff.NonReloadable)}
	}
	return ReloadResult{Diff: diff, Applied: true}
}

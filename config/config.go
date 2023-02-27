package config

import (
	"SyncWatcher/lib/types"
	"encoding/json"
	"fmt"
)

type Config struct {
	Log     ConfigLog                     `json:"log,omitempty"`
	Watcher types.Listable[ConfigWatcher] `json:"watcher,omitempty"`
}

type ConfigLog struct {
	File  string `json:"file,omitempty"`
	Debug bool   `json:"debug,omitempty"`
}

type ConfigWatcher struct {
	Path     string                 `json:"path,omitempty"`
	Monitor  types.Listable[string] `json:"monitor,omitempty"`
	FirstRun bool                   `json:"first-run,omitempty"`
	Mode     string                 `json:"mode,omitempty"`
	Rule     types.Listable[string] `json:"rule,omitempty"`
	Script   types.Listable[string] `json:"script,omitempty"`
}

type _config Config

func (c *Config) UnmarshalJSON(content []byte) error {
	var _c _config
	if err := json.Unmarshal(content, &_c); err != nil {
		return err
	}
	if _c.Watcher == nil || len(_c.Watcher) == 0 {
		return fmt.Errorf("no watcher defined")
	}
	m := make(map[string]int)
	for k, w := range _c.Watcher {
		if _, ok := m[fmt.Sprintf("path:%s", w.Path)]; ok {
			return fmt.Errorf("duplicate watcher path: %s", w.Path)
		} else {
			m[fmt.Sprintf("path:%s", w.Path)]++
		}
		switch w.Mode {
		case "include":
			_c.Watcher[k].Mode = "include"
		case "", "exclude":
			_c.Watcher[k].Mode = "exclude"
		default:
			return fmt.Errorf("invalid watcher mode: %s", w.Mode)
		}
		if w.Rule != nil && len(w.Rule) > 0 {
			m := make(map[string]int)
			for _, r := range w.Rule {
				if _, ok := m[fmt.Sprintf("rule:%s", r)]; ok {
					return fmt.Errorf("duplicate watcher rule: %s", r)
				} else {
					m[fmt.Sprintf("rule:%s", r)]++
				}
			}
		}
		if w.Script == nil || len(w.Script) == 0 {
			return fmt.Errorf("no script defined for watcher: %s", w.Path)
		}
		m := make(map[string]int)
		for _, s := range w.Script {
			if _, ok := m[fmt.Sprintf("script:%s", s)]; ok {
				return fmt.Errorf("duplicate watcher script: %s", s)
			} else {
				m[fmt.Sprintf("script:%s", s)]++
			}
		}
	}
	*c = Config(_c)
	return nil
}

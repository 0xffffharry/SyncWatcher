package config

import (
	"encoding/json"
	"fmt"
	"os"
)

const (
	DefaultTerminal = "/bin/bash"
)

type Config struct {
	Watchers []Watcher `json:"watcher"`
}

type Watcher struct {
	Path        string   `json:"path"`
	IgnorePaths []string `json:"ignore_paths"`
	Ops         []string `json:"ops"`
	Script      []Script `json:"script"`
	RunFirst    bool     `json:"run_first"`
}

type Script struct {
	Name string   `json:"name"`
	Args []string `json:"args"`
}

func Parse(filename string) (*Config, error) {
	var config Config
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	decode := json.NewDecoder(f)
	decode.DisallowUnknownFields()
	err = decode.Decode(&config)
	f.Close()
	if err != nil {
		return nil, err
	}
	if config.Watchers == nil || len(config.Watchers) == 0 {
		return nil, fmt.Errorf("no watcher found")
	} else {
		for k, v := range config.Watchers {
			if v.Path == "" {
				return nil, fmt.Errorf("path is empty")
			} else {
				p, err := os.Stat(v.Path)
				if err != nil {
					return nil, fmt.Errorf("open path fail: %s", err)
				}
				if !p.IsDir() {
					return nil, fmt.Errorf("path must be a directory")
				}
			}
			if v.Script == nil || len(v.Script) == 0 {
				return nil, fmt.Errorf("script is empty")
			} else {
				for _, s := range v.Script {
					if s.Name == "" {
						return nil, fmt.Errorf("script name is empty")
					}
				}
			}
			if v.Ops != nil && len(v.Ops) > 0 {
				opsCacheMap := make(map[string]bool)
				opsNew := make([]string, 0)
				for _, o := range v.Ops {
					switch o {
					case "write", "create", "remove", "rename", "chmod":
						if _, ok := opsCacheMap[o]; !ok {
							opsCacheMap[o] = true
							opsNew = append(opsNew, o)
						}
					default:
						return nil, fmt.Errorf("invalid ops: %s", o)
					}
				}
				config.Watchers[k].Ops = opsNew
			}
		}
	}
	return &config, nil
}

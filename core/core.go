package core

import (
	"SyncWatcher/config"
	"SyncWatcher/lib/command"
	"SyncWatcher/lib/constant"
	"SyncWatcher/lib/log"
	"context"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"io/fs"
	"path/filepath"
	"regexp"
	"sync"
	"sync/atomic"
	"time"
)

func New(watcher *config.ConfigWatcher) (*Watcher, error) {
	w := &Watcher{}
	if watcher.Path == "" {
		return nil, fmt.Errorf("path is empty")
	}
	w.path = watcher.Path
	if watcher.Mode == "" {
		return nil, fmt.Errorf("mode is empty")
	}
	w.firstRun = watcher.FirstRun
	if watcher.Monitor == nil || len(watcher.Monitor) == 0 {
		w.monitor = constant.DefaultMonitor
	} else {
		w.monitor = make([]fsnotify.Op, 0)
		for _, m := range watcher.Monitor {
			switch m {
			case "create":
				w.monitor = append(w.monitor, fsnotify.Create)
			case "write":
				w.monitor = append(w.monitor, fsnotify.Write)
			case "remove":
				w.monitor = append(w.monitor, fsnotify.Remove)
			case "rename":
				w.monitor = append(w.monitor, fsnotify.Rename)
			default:
				return nil, fmt.Errorf("invalid monitor: %s", m)
			}
		}
	}
	switch watcher.Mode {
	case "include":
		w.mode = watcher.Mode
	case "exclude":
		w.mode = watcher.Mode
	default:
		return nil, fmt.Errorf("invalid mode: %s", watcher.Mode)
	}
	if watcher.Rule != nil && len(watcher.Rule) > 0 {
		w.rule = make([]*regexp.Regexp, 0)
		for _, r := range watcher.Rule {
			rr, err := regexp.Compile(r)
			if err != nil {
				return nil, fmt.Errorf("invalid rule: %s", r)
			}
			w.rule = append(w.rule, rr)
		}
	}
	if watcher.Script == nil || len(watcher.Script) == 0 {
		return nil, fmt.Errorf("script is empty")
	} else {
		w.script = make([]*command.Command, 0)
		for _, s := range watcher.Script {
			if s.Path == "" {
				return nil, fmt.Errorf("script command is empty")
			}
			w.script = append(w.script, command.New(s.Path, time.Duration(s.Timeout)))
		}
	}
	return w, nil
}

func (w *Watcher) RunWithContext(ctx context.Context, logger *log.Logger) {
	if w.ctx == nil {
		if ctx == nil {
			ctx = context.Background()
		}
		w.ctx = ctx
	}
	if w.logger == nil {
		if logger == nil {
			logger = log.NewLogger(nil, nil)
		}
		w.logger = logger
	}
	w.logger.Info("watcher", fmt.Sprintf("start watcher `%s`", w.path))
	defer w.logger.Info("watcher", fmt.Sprintf("stop watcher `%s`", w.path))
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		w.logger.Fatal("watcher", fmt.Sprintf("create watcher failed: %s", err))
		return
	}
	defer watcher.Close()
	err = filepath.WalkDir(w.path, func(path string, d fs.DirEntry, err error) error {
		simplePath, err := filepath.Rel(w.path, path)
		if err != nil {
			return fmt.Errorf("get simple path fail: %s", err)
		}
		if w.rule != nil && len(w.rule) > 0 {
			match := atomic.Bool{}
			wg := sync.WaitGroup{}
			for _, r := range w.rule {
				wg.Add(1)
				go func(r *regexp.Regexp) {
					defer wg.Done()
					if r.MatchString(simplePath) {
						match.Store(true)
					}
				}(r)
			}
			wg.Wait()
			if match.Load() {
				if w.mode == "include" {
					err := watcher.Add(path)
					if err != nil {
						return fmt.Errorf("add path `%s` fail: %s", path, err)
					}
				} else {
					w.logger.Info("watcher", fmt.Sprintf("skip path `%s`", path))
				}
			} else {
				if w.mode == "exclude" {
					err := watcher.Add(path)
					if err != nil {
						return fmt.Errorf("add path `%s` fail: %s", path, err)
					}
				} else {
					w.logger.Info("watcher", fmt.Sprintf("skip path `%s`", path))
				}
			}
		} else {
			err := watcher.Add(path)
			if err != nil {
				return fmt.Errorf("add path `%s` fail: %s", path, err)
			}
		}
		return nil
	})
	if err != nil {
		w.logger.Fatal("watcher", err.Error())
		return
	}
	if w.firstRun {
		w.logger.Info("watcher", fmt.Sprintf("first script run"))
		for _, s := range w.script {
			s = s.Clone()
			s.SetEnv("syncdir", w.path)
			_, _, err := s.RunWithContext(w.ctx)
			if err != nil {
				logger.Error("watcher", fmt.Sprintf("run script `%s` failed: %s", s.String(), err))
			} else {
				logger.Info("watcher", fmt.Sprintf("run script `%s` success", s.String()))
			}
		}
		w.logger.Info("watcher", fmt.Sprintf("first script run success"))
	}
	w.callValue = &atomic.Bool{}
	w.callValue.Store(false)
	go w.call()
	for {
		select {
		case op := <-watcher.Events:
			switch {
			case op.Has(fsnotify.Create):
				err := watcher.Add(op.Name)
				if err != nil {
					w.logger.Warn("watcher", fmt.Sprintf("add path `%s` fail: %s", op.Name, err))
				}
			case op.Has(fsnotify.Rename):
			case op.Has(fsnotify.Remove):
				watcher.Remove(op.Name)
			}
			w.logger.Debug("watcher", fmt.Sprintf("event: %s path: %s", op.Op.String(), op.Name))
			tag := false
			for _, k := range w.monitor {
				if op.Has(k) {
					tag = true
					break
				}
			}
			if tag {
				w.callValue.Store(true)
			}
		case <-w.ctx.Done():
			return
		}
	}
}

func (w *Watcher) call() {
	for {
		select {
		case <-time.After(100 * time.Millisecond):
			if !w.callValue.Load() {
				continue
			}
			w.callValue.Store(false)
			w.logger.Info("watcher", fmt.Sprintf("run script"))
			for _, s := range w.script {
				s = s.Clone()
				s.SetEnv("syncdir", w.path)
				_, _, err := s.RunWithContext(w.ctx)
				if err != nil {
					w.logger.Error("watcher", fmt.Sprintf("run script `%s` failed: %s", s.String(), err))
				} else {
					w.logger.Info("watcher", fmt.Sprintf("run script `%s` success", s.String()))
				}
			}
		case <-w.ctx.Done():
			return
		}
	}
}

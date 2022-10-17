package watcher

import (
	"SyncWatcher/log"
	"context"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
)

func (c *Config) NewWatcher() error {
	if c.Path != "" {
		_, err := os.Stat(c.Path)
		if err != nil {
			return fmt.Errorf("open path fail: %s", err)
		}
	} else {
		return fmt.Errorf("path not found")
	}
	c.config = config{
		Global: Global{},
	}
	c.config.Path = c.Path
	c.config.RunFirst = c.RunFirst
	if c.Ctx == nil {
		c.config.Global.Ctx = context.Background()
	} else {
		c.config.Global.Ctx = c.Ctx
	}
	if c.Logger == nil {
		c.config.Global.Logger = log.New().SetOutput(os.Stdout)
	} else {
		c.config.Global.Logger = c.Logger
	}
	if c.CaptureOperations == nil || len(c.CaptureOperations) == 0 {
		c.config.CaptureOperations = defaultCaptureOperations()
	} else {
		c.config.CaptureOperations = make(map[fsnotify.Op]bool)
		for _, v := range c.CaptureOperations {
			switch v {
			case string(Write):
				c.config.CaptureOperations[fsnotify.Write] = true
			case string(Create):
				c.config.CaptureOperations[fsnotify.Create] = true
			case string(Remove):
				c.config.CaptureOperations[fsnotify.Remove] = true
			case string(Rename):
				c.config.CaptureOperations[fsnotify.Rename] = true
			case string(Chmod):
				c.config.CaptureOperations[fsnotify.Chmod] = true
			}
		}
	}
	if c.Script != nil && len(c.Script) > 0 {
		c.config.Script = c.Script
	}
	if c.IgnorePaths != nil && len(c.IgnorePaths) > 0 {
		c.config.IgnorePaths = make([]*regexp.Regexp, 0)
		for _, v := range c.IgnorePaths {
			r, err := regexp.Compile(v)
			if err != nil {
				return fmt.Errorf("compile ignore path fail: %s", err)
			}
			c.config.IgnorePaths = append(c.config.IgnorePaths, r)
		}
	} else {
		c.config.IgnorePaths = nil
	}
	return nil
}

func (c *Config) Run() {
	c.config.run()
}

func (c *config) run() {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		c.Global.Logger.Fatalln(log.Fatal, fmt.Sprintf("create new watcher fail: %s", err))
		return
	}
	err = filepath.WalkDir(c.Path, func(path string, d fs.DirEntry, err error) error {
		if c.matchIgnorePath(path, false) {
			c.Global.Logger.Println(log.Info, fmt.Sprintf("ignore path: %s", path))
			return nil
		}
		return w.Add(path)
	})
	if err != nil {
		c.Global.Logger.Fatalln(log.Fatal, fmt.Sprintf("add path fail: %s", err))
		return
	}
	if c.RunFirst {
		c.runScript(fsnotify.Event{})
	}
	c.WatchChan = make(chan fsnotify.Event, 1024)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.listenEvents()
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case p := <-w.Events:
				if c.CaptureOperations[p.Op] {
					c.matchIgnorePath(p.Name, true)
					c.Global.Logger.Println(log.Info, fmt.Sprintf("path: `%s` event: %s", p.Name, p.Op.String()))
					switch {
					case p.Op.Has(fsnotify.Rename):
					case p.Op.Has(fsnotify.Remove):
						_ = w.Remove(p.Name)
					case p.Op.Has(fsnotify.Create):
						_ = w.Add(p.Name)
					}
					c.WatchChan <- p
				}
			case <-c.Global.Ctx.Done():
				return
			}
		}
	}()
	wg.Wait()
	close(c.WatchChan)
	c.Global.Logger.Println(log.Info, "watcher exit")
}

func (c *config) matchIgnorePath(path string, abs bool) bool {
	if abs {
		path, err := filepath.Rel(c.Path, path)
		if err != nil {
			return false
		}
		path = filepath.Clean(path)
	}
	if c.IgnorePaths != nil && len(c.IgnorePaths) > 0 {
		Match := atomic.Bool{}
		Match.Store(false)
		wg := sync.WaitGroup{}
		for _, v := range c.IgnorePaths {
			wg.Add(1)
			go func(r *regexp.Regexp, name string) {
				defer wg.Done()
				if r.MatchString(name) {
					Match.Store(true)
				}
			}(v, path)
		}
		wg.Wait()
		return Match.Load()
	} else {
		return false
	}
}

func (c *config) listenEvents() {
	runLock := atomic.Bool{}
	runLock.Store(false)
	run := func(e fsnotify.Event) {
		if runLock.Load() {
			return
		}
		runLock.Store(true)
		defer runLock.Store(false)
		c.runScript(e)
	}
	for {
		select {
		case p := <-c.WatchChan:
			run(p)
		case <-c.Global.Ctx.Done():
			return
		}
	}
}

func (c *config) runScript(e fsnotify.Event) {
	if c.Script != nil && len(c.Script) > 0 {
		for _, v := range c.Script {
			var cmd *exec.Cmd
			if v.Args != nil && len(v.Args) > 0 {
				cmd = exec.CommandContext(c.Global.Ctx, v.Name, v.Args...)
			} else {
				cmd = exec.CommandContext(c.Global.Ctx, v.Name)
			}
			var (
				Output    = io.Discard // Output to NullDevice
				ErrOutput = io.Discard // Output to NullDevice
			)
			cmd.Stdout = Output
			cmd.Stderr = ErrOutput
			if e.Name != "" && e.Op.String() != "[no events]" {
				cmd.Env = append(os.Environ(), fmt.Sprintf("WATCHER_NAME=%s", e.Name))
				cmd.Env = append(cmd.Env, fmt.Sprintf("WATCHER_EVENT=%s", e.Op.String()))
			}
			c.Global.Logger.Println(log.Info, fmt.Sprintf("run script: %s %s", v.Name, strings.Join(v.Args, " ")))
			err := cmd.Run()
			if err != nil {
				c.Global.Logger.Println(log.Error, fmt.Sprintf("run script `%s %s` fail: %s", v.Name, strings.Join(v.Args, " "), err))
				continue
			}
			c.Global.Logger.Println(log.Info, fmt.Sprintf("run script `%s %s` success", v.Name, strings.Join(v.Args, " ")))
		}
	}
}

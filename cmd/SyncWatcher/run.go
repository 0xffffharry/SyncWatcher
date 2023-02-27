package main

import (
	"SyncWatcher/config"
	"SyncWatcher/core"
	"SyncWatcher/lib/constant"
	"SyncWatcher/lib/log"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func run() {
	configContent, err := os.ReadFile(paramConfig)
	if err != nil {
		fmt.Println(fmt.Sprintf("read config file failed: %s", err))
		os.Exit(1)
	}
	var cfg config.Config
	err = json.Unmarshal(configContent, &cfg)
	if err != nil {
		fmt.Println(fmt.Sprintf("parse config file failed: %s", err))
		os.Exit(1)
	}
	watchers := make([]*core.Watcher, 0)
	for _, watcherCfg := range cfg.Watcher {
		watcher, err := core.New(&watcherCfg)
		if err != nil {
			fmt.Println(fmt.Sprintf("create watcher failed: %s", err))
			os.Exit(1)
		}
		watchers = append(watchers, watcher)
	}
	logger := log.NewLogger(os.Stdout, nil)
	if cfg.Log.File != "" {
		os.Remove(cfg.Log.File)
		f, err := os.Create(cfg.Log.File)
		if err != nil {
			fmt.Println(fmt.Sprintf("open log file failed: %s", err))
			os.Exit(1)
		}
		defer f.Close()
		logger.SetOutput(f)
	}
	logger.Info("global", fmt.Sprintf("SyncWatcher %s", constant.Version))
	defer logger.Info("global", "Bye!!")
	if cfg.Log.Debug {
		logger.SetDebug(cfg.Log.Debug)
		logger.Debug("global", "debug mode enabled")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go listenSignal(cancel, logger)
	wg := sync.WaitGroup{}
	for _, watcher := range watchers {
		wg.Add(1)
		go func(watcher *core.Watcher) {
			defer wg.Done()
			watcher.RunWithContext(ctx, logger)
		}(watcher)
	}
	wg.Wait()
}

func listenSignal(cancelFunc context.CancelFunc, logger *log.Logger) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c
	logger.Warn("global", "receive signal, exit")
	cancelFunc()
}

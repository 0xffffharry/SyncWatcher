package main

import (
	"SyncWatcher/config"
	"SyncWatcher/log"
	"SyncWatcher/watcher"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

const (
	AppName    = "SyncWatcher"
	AppVersion = "v2.0.1-alpha"
)

func main() {
	c := flag.String("c", "config.json", "config file")
	v := flag.Bool("v", false, "show version")
	h := flag.Bool("h", false, "show help")
	flag.Parse()
	if *v {
		fmt.Println(fmt.Sprintf("%s %s", AppName, AppVersion))
		return
	}
	if *h {
		flag.Usage()
		return
	}
	logger := log.New().SetOutput(os.Stdout)
	logger.Println(log.Info, fmt.Sprintf("%s %s", AppName, AppVersion))
	logger.Println(log.Info, "Running...")
	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	runWork := sync.WaitGroup{}
	for {
		ctx, cancel := context.WithCancel(context.Background())
		runMain(ctx, logger, *c, &runWork)
		for {
			ContinueTag := false
			osSignal := <-osSignals
			switch osSignal {
			case syscall.SIGHUP:
				logger.Println(log.Info, "receive SIGHUP signal, reload config file")
				cancel()
				runWork.Wait()
				ContinueTag = true
			default:
				logger.Println(log.Info, fmt.Sprintf("receive %s signal, exit", osSignal))
				cancel()
				runWork.Wait()
				return
			}
			if ContinueTag {
				break
			}
		}
	}
}

func runMain(ctx context.Context, logger *log.Logger, filename string, runWork *sync.WaitGroup) {
	c, err := config.Parse(filename)
	if err != nil {
		logger.Fatalln(log.Fatal, fmt.Sprintf("parse config file fail: %s", err))
		return
	}
	for _, v := range c.Watchers {
		watcherConfig := watcher.Config{
			Path:              v.Path,
			CaptureOperations: v.Ops,
			IgnorePaths:       v.IgnorePaths,
			RunFirst:          v.RunFirst,
			Logger:            logger,
			Ctx:               ctx,
		}
		if v.Script != nil && len(v.Script) > 0 {
			watcherConfig.Script = make([]watcher.Script, 0)
			for _, script := range v.Script {
				watcherConfig.Script = append(watcherConfig.Script, watcher.Script{
					Name: script.Name,
					Args: script.Args,
				})
			}
		}
		err = watcherConfig.NewWatcher()
		if err != nil {
			logger.Fatalln(log.Fatal, fmt.Sprintf("create watcher fail: %s", err))
			return
		}
		logger.Println(log.Info, fmt.Sprintf("watcher run: %s", watcherConfig.Path))
		runWork.Add(1)
		go func(watcherConfig watcher.Config, runWork *sync.WaitGroup) {
			defer runWork.Done()
			watcherConfig.Run()
		}(watcherConfig, runWork)
	}
}

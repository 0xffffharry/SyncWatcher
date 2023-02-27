package core

import (
	"SyncWatcher/lib/command"
	"SyncWatcher/lib/log"
	"context"
	"github.com/fsnotify/fsnotify"
	"regexp"
	"sync/atomic"
)

type Watcher struct {
	path     string
	monitor  []fsnotify.Op
	firstRun bool
	mode     string
	rule     []*regexp.Regexp
	script   []*command.Command
	//
	ctx    context.Context
	logger *log.Logger
	//
	callValue *atomic.Bool
}

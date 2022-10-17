package watcher

import (
	"SyncWatcher/log"
	"context"
	"github.com/fsnotify/fsnotify"
	"regexp"
)

type CaptureOperation string

const (
	Write  CaptureOperation = "write"
	Create CaptureOperation = "create"
	Remove CaptureOperation = "remove"
	Rename CaptureOperation = "rename"
	Chmod  CaptureOperation = "chmod"
)

type Config struct {
	Path              string   // 路径
	CaptureOperations []string // 捕获操作
	Script            []Script // 脚本
	IgnorePaths       []string // 忽略路径
	RunFirst          bool     // 是否首次运行
	//
	Logger *log.Logger
	Ctx    context.Context
	//
	config config
}

type Global struct {
	Logger *log.Logger
	Ctx    context.Context
}

type config struct {
	Global Global
	//
	Path              string
	CaptureOperations map[fsnotify.Op]bool
	Script            []Script
	IgnorePaths       []*regexp.Regexp
	RunFirst          bool
	//
	WatchChan chan fsnotify.Event
}

type Script struct {
	Name string
	Args []string
}

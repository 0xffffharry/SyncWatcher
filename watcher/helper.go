package watcher

import (
	"github.com/fsnotify/fsnotify"
)

func defaultCaptureOperations() map[fsnotify.Op]bool {
	return map[fsnotify.Op]bool{
		fsnotify.Write:  true,
		fsnotify.Create: true,
		fsnotify.Rename: true,
		fsnotify.Remove: true,
	}
}

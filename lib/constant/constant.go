package constant

import "github.com/fsnotify/fsnotify"

const Version = "v3.0.0-alpha1"

var DefaultMonitor = []fsnotify.Op{fsnotify.Create, fsnotify.Write, fsnotify.Remove, fsnotify.Rename}

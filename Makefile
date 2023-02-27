NAME = SyncWatcher
PARAMS = -v -trimpath -ldflags "-s -w -buildid="
MAIN = ./cmd/SyncWatcher

build:
	go build $(PARAMS) $(MAIN)

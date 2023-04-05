package command

import "time"

type Command struct {
	Name    string
	Timeout time.Duration
	Env     []string
}

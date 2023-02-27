package command

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func New(name string) *Command {
	return &Command{
		Name: name,
	}
}

func (c *Command) Clone() *Command {
	return &Command{
		Name: c.Name,
		Env:  c.Env,
	}
}

func (c *Command) String() string {
	name := c.Name
	for _, v := range c.Env {
		kv := strings.SplitN(v, "=", 2)
		name = strings.ReplaceAll(name, fmt.Sprintf("${%s}", kv[0]), kv[1])
		name = strings.ReplaceAll(name, fmt.Sprintf("$%s", kv[0]), kv[1])
	}
	return name
}

func (c *Command) SetEnv(key, value string) {
	if c.Env == nil {
		c.Env = make([]string, 0)
	}
	c.Env = append(c.Env, key+"="+value)
}

func (c *Command) Run() ([]byte, []byte, error) {
	return c.RunWithContext(context.Background())
}

func (c *Command) RunWithContext(ctx context.Context) ([]byte, []byte, error) {
	lineSlice := strings.SplitN(c.Name, " ", 2)
	cmd := exec.CommandContext(ctx, lineSlice[0], lineSlice[1:]...)
	var (
		stdOutput = bytes.NewBuffer(nil)
		errOutput = bytes.NewBuffer(nil)
	)
	cmd.Stdout = stdOutput
	cmd.Stderr = errOutput
	if cmd.Env == nil {
		cmd.Env = make([]string, 0)
	}
	cmd.Env = append(cmd.Env, c.Env...)
	err := cmd.Run()
	return stdOutput.Bytes(), errOutput.Bytes(), err
}

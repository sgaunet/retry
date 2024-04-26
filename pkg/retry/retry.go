package retry

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/go-andiamo/splitter"
)

type retry struct {
	cmd       string
	sleep     func()
	tries     uint
	condition ConditionRetryer
}

type ConditionRetryer interface {
	GetCtx() context.Context
	IsLimitReached() bool
	StartTry()
	EndTry()
}

func NewRetry(cmd string, condition ConditionRetryer) (*retry, error) {
	r := &retry{
		cmd:       cmd,
		sleep:     nil,
		condition: condition,
	}
	if r.condition == nil {
		return nil, fmt.Errorf("condition is nil")
	}
	return r, nil
}

func (r *retry) SetSleep(sleep func()) {
	r.sleep = sleep
}

func (r *retry) Run() error {
	var (
		err error
	)
	for r.condition.GetCtx().Err() == nil && !r.condition.IsLimitReached() {
		if r.condition != nil {
			r.condition.StartTry()
		}
		r.tries++
		err = execCommand(r.condition.GetCtx(), r.cmd)
		if r.condition != nil {
			r.condition.EndTry()
		}
		if err == nil {
			break
		}
		if r.sleep != nil {
			r.sleep()
		}
	}
	return err
}

func execCommand(ctx context.Context, cmd string) error {
	commandSplitter, _ := splitter.NewSplitter(' ', splitter.SingleQuotes, splitter.DoubleQuotes)
	trimmer := splitter.Trim("'\"")
	splitCmd, _ := commandSplitter.Split(cmd, trimmer)
	if len(splitCmd) == 0 {
		return nil
	}
	c := exec.CommandContext(ctx, splitCmd[0], splitCmd[1:]...) //nolint:gosec

	stderr, err := c.StderrPipe()
	if err != nil {
		return err
	}

	stdout, err := c.StdoutPipe()
	if err != nil {
		return err
	}

	// print output in real time (both stdout and stderr)
	go printOutput(stdout, os.Stdout)
	go printOutput(stderr, os.Stderr)

	err = c.Start()
	if err != nil {
		return err
	}
	err = c.Wait()
	return err
}

func printOutput(r io.Reader, w io.Writer) {
	var reader = bufio.NewScanner(r)
	for reader.Scan() {
		_, _ = fmt.Fprintln(w, reader.Text())
	}
}

// Command lab is the thin client.
//
// It resolves a verb, sends one message, and does exactly one of: print,
// prompt, or replace itself with a console. Everything expensive lives in
// labd, which is why this costs ~2ms to run.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/dmbdata101/lab/internal/client"
	"github.com/dmbdata101/lab/internal/paths"
	"github.com/dmbdata101/lab/internal/picker"
	"github.com/dmbdata101/lab/internal/proto"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		if errors.Is(err, picker.ErrCancelled) {
			os.Exit(130)
		}
		fmt.Fprintln(os.Stderr, "lab: "+err.Error())
		os.Exit(1)
	}
}

func run(args []string) error {
	verb := "help"
	if len(args) > 0 {
		verb = args[0]
		args = args[1:]
	} else {
		args = nil
	}

	c := &client.Client{Socket: paths.Socket(), AutoStart: true}
	resp, err := c.Send(&proto.Request{Verb: verb, Args: args})
	if err != nil {
		return err
	}

	// A picker round trip: the daemon never touches the terminal, it just
	// says what it needs and we ask.
	if len(resp.Choices) > 0 {
		id, err := picker.Choose(resp.PickTitle, resp.Choices)
		if err != nil {
			return err
		}
		resp, err = c.Send(&proto.Request{Verb: resp.PickVerb, Args: []string{id}})
		if err != nil {
			return err
		}
	}

	if !resp.OK {
		return errors.New(resp.Error)
	}
	if os.Getenv("LAB_TIME") != "" {
		fmt.Fprintf(os.Stderr, "[%.1fms]\n", resp.ElapsedMS)
	}

	if len(resp.Exec) > 0 {
		return attach(resp.Exec)
	}
	if resp.Output != "" {
		fmt.Println(resp.Output)
	}
	return nil
}

// attach replaces this process with the console command. Exec rather than a
// subprocess so signals, job control, and the TTY all behave exactly as if
// the command had been typed directly.
func attach(argv []string) error {
	bin, err := exec.LookPath(argv[0])
	if err != nil {
		return fmt.Errorf("console command %q not found: %w", argv[0], err)
	}
	return syscall.Exec(bin, argv, os.Environ())
}

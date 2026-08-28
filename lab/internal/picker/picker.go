// Package picker turns "you didn't give me an argument" into a choice.
//
// This is the interaction rule the whole vocabulary rests on: every verb run
// bare opens a picker, so only verbs ever have to be remembered and the fast
// path (typing the argument) is learned by using the slow one.
package picker

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/dmbdata101/lab/internal/proto"
)

// ErrCancelled is returned when the user backs out.
var ErrCancelled = errors.New("cancelled")

// Choose presents choices and returns the chosen ID.
//
// fzf is used when present because it is better than anything worth writing
// here; the builtin exists so the tool has no hard dependency.
func Choose(title string, choices []proto.Choice) (string, error) {
	switch len(choices) {
	case 0:
		return "", errors.New("nothing to choose from")
	case 1:
		// One option is not a choice.
		return choices[0].ID, nil
	}
	if _, err := exec.LookPath("fzf"); err == nil {
		if id, err := chooseFzf(title, choices); !errors.Is(err, errNoFzf) {
			return id, err
		}
	}
	return chooseBuiltin(title, choices)
}

var errNoFzf = errors.New("fzf unusable")

func chooseFzf(title string, choices []proto.Choice) (string, error) {
	var in strings.Builder
	for i, c := range choices {
		// Index prefix keeps the mapping exact even if two labels collide.
		fmt.Fprintf(&in, "%d\t%s\n", i, c.Label)
	}
	cmd := exec.Command("fzf",
		"--prompt", title+" > ",
		"--height", "40%", "--reverse",
		"--with-nth", "2..", "--delimiter", "\t",
		"--select-1", "--exit-0")
	cmd.Stdin = strings.NewReader(in.String())
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 130 {
			return "", ErrCancelled // Esc / Ctrl-C
		}
		return "", errNoFzf
	}
	idx, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\t")
	i, err := strconv.Atoi(idx)
	if err != nil || i < 0 || i >= len(choices) {
		return "", errNoFzf
	}
	return choices[i].ID, nil
}

// chooseBuiltin reads from the controlling terminal, so the picker still works
// when stdout is piped.
func chooseBuiltin(title string, choices []proto.Choice) (string, error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return "", fmt.Errorf("%s: no argument given and no terminal to ask on", title)
	}
	defer tty.Close()

	fmt.Fprintf(tty, "%s\n", title)
	for i, c := range choices {
		fmt.Fprintf(tty, "  %2d  %s\n", i+1, c.Label)
	}
	fmt.Fprint(tty, "> ")

	line, err := bufio.NewReader(tty).ReadString('\n')
	if err != nil {
		return "", ErrCancelled
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return "", ErrCancelled
	}
	if n, err := strconv.Atoi(line); err == nil {
		if n < 1 || n > len(choices) {
			return "", fmt.Errorf("no choice %d", n)
		}
		return choices[n-1].ID, nil
	}
	return matchPrefix(line, choices)
}

// matchPrefix accepts a substring as long as it is unambiguous — typing "r1"
// should be enough when nothing else contains it.
func matchPrefix(s string, choices []proto.Choice) (string, error) {
	s = strings.ToLower(s)
	var hits []proto.Choice
	for _, c := range choices {
		if strings.Contains(strings.ToLower(c.ID), s) || strings.Contains(strings.ToLower(c.Label), s) {
			hits = append(hits, c)
		}
	}
	switch len(hits) {
	case 0:
		return "", fmt.Errorf("nothing matches %q", s)
	case 1:
		return hits[0].ID, nil
	default:
		names := make([]string, 0, len(hits))
		for _, h := range hits {
			names = append(names, h.ID)
		}
		return "", fmt.Errorf("%q is ambiguous: %s", s, strings.Join(names, " "))
	}
}

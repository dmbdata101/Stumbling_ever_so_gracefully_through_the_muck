// Package proto defines the wire format between the lab client and labd.
//
// One JSON object per line, request then response, connection closed after.
// Deliberately boring: the client must stay tiny and start in ~2ms, so it
// depends on nothing but encoding/json and net.
package proto

// Request is what the client sends.
type Request struct {
	Verb string   `json:"verb"`
	Args []string `json:"args"`
	Cwd  string   `json:"cwd,omitempty"`
	TTY  bool     `json:"tty,omitempty"`
}

// Choice is one option in a picker prompt.
type Choice struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Response is what labd sends back.
//
// Exactly one of Output, Exec, or Choices is meaningful on success:
//
//	Output  - print it and exit
//	Exec    - replace this process with that command (console attach); the
//	          daemon never owns a TTY, the client does
//	Choices - the verb needs an argument the user did not supply. Show a
//	          picker, then re-send Verb=PickVerb with the chosen ID. This is
//	          how "every verb with no arguments drops into a picker" works
//	          without the daemon ever touching the terminal.
type Response struct {
	OK        bool     `json:"ok"`
	Output    string   `json:"output,omitempty"`
	Error     string   `json:"error,omitempty"`
	Exec      []string `json:"exec,omitempty"`
	Choices   []Choice `json:"choices,omitempty"`
	PickVerb  string   `json:"pick_verb,omitempty"`
	PickTitle string   `json:"pick_title,omitempty"`
	ElapsedMS float64  `json:"elapsed_ms"`
}

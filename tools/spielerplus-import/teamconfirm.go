package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// ErrTeamNotConfirmed is returned by confirmTeam when the run must not
// proceed: either the operator declined the interactive prompt, or
// SPIELERPLUS_EXPECTED_TEAM_NAME was set and didn't match the detected team.
var ErrTeamNotConfirmed = fmt.Errorf("spielerplus-import: active SpielerPlus team not confirmed")

// confirmTeam guards against an account that manages more than one
// SpielerPlus team having the wrong one active when its session cookie was
// captured (see spielerplus.Client.FetchActiveTeamName's doc comment).
//
// If expected is non-empty (SPIELERPLUS_EXPECTED_TEAM_NAME), detected is
// compared against it directly and the call returns without prompting -
// for non-interactive/repeatable runs once an operator has confirmed the
// team name once. Otherwise it prints detected and asks for an explicit
// "y"/"yes" on in, aborting on anything else (including EOF, e.g. no TTY
// attached and the env var wasn't set either).
func confirmTeam(detected, expected string, in io.Reader, out io.Writer) error {
	if expected != "" {
		if !strings.EqualFold(strings.TrimSpace(detected), strings.TrimSpace(expected)) {
			return fmt.Errorf("%w: SPIELERPLUS_EXPECTED_TEAM_NAME is %q but the active SpielerPlus team is %q - switch to the right team at https://www.spielerplus.de/site/select-team, capture a fresh session cookie, and try again", ErrTeamNotConfirmed, expected, detected)
		}
		fmt.Fprintf(out, "Active SpielerPlus team %q matches SPIELERPLUS_EXPECTED_TEAM_NAME, proceeding.\n", detected)
		return nil
	}

	fmt.Fprintf(out, "Detected active SpielerPlus team: %q\n", detected)
	fmt.Fprintf(out, "This is the team every SpielerPlus page this tool reads will be scoped to - if your account manages more than one, double check this is the right one (switch at https://www.spielerplus.de/site/select-team and re-capture the session cookie if not).\n")
	fmt.Fprint(out, "Continue with this team? [y/N]: ")

	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && err != io.EOF {
		return fmt.Errorf("spielerplus-import: read confirmation: %w", err)
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	if answer != "y" && answer != "yes" {
		return fmt.Errorf("%w: not confirmed (set SPIELERPLUS_EXPECTED_TEAM_NAME to skip this prompt on repeat runs)", ErrTeamNotConfirmed)
	}
	return nil
}

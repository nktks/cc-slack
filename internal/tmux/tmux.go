package tmux

import (
	"fmt"
	"os/exec"
)

// SendKeys sends a message to the specified tmux target pane.
// The message and Enter keystroke are sent as separate send-keys commands.
func SendKeys(target, message string) error {
	if err := exec.Command("tmux", "send-keys", "-t", target, message).Run(); err != nil {
		return fmt.Errorf("tmux send-keys message: %w", err)
	}
	if err := exec.Command("tmux", "send-keys", "-t", target, "Enter").Run(); err != nil {
		return fmt.Errorf("tmux send-keys Enter: %w", err)
	}
	return nil
}

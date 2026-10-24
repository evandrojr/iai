package notify

import (
	"os/exec"
)

func Send(title, body string) {
	cmd := exec.Command("notify-send",
		"--urgency=critical",
		"--app-name=iai",
		title,
		body,
	)
	_ = cmd.Run()
}

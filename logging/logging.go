package logging

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os/exec"
	"strings"
)

type CmdWriter struct {
	Host string
}

func (cmdWriter CmdWriter) Write(inputBytes []byte) (n int, err error) {
	msg := string(inputBytes)
	// A newline will always be included as suffix. Unless removed, this will
	// lead to always printing an empty line at the end.
	msg = strings.TrimSuffix(msg, "\n")
	lines := strings.Split(msg, "\n")

	for _, line := range lines {
		log.Info().Str("Host", cmdWriter.Host).Msg(line)
	}
	return len(inputBytes), nil // This is obviously a lie..
}

func LogCmd(host string, cmd *exec.Cmd) {
	writer := CmdWriter{Host: host}

	cmd.Stdout = writer
	cmd.Stderr = writer

	LogCommand(host, cmd)
}

func LogCommand(host string, cmd *exec.Cmd) {
	// FIXME: This seems to log the executable twice, one with the full path, and one as name only
	zLogCmd := zerolog.Arr().Str(cmd.Path)
	first := true
	for _, arg := range cmd.Args {
		// First arg is the name of the command to run:
		// Skip it, since cmd.Path is the resolved path of the command,
		// and is added already.
		if first {
			first = false
			continue
		}

		zLogCmd.Str(arg)
	}

	log.Info().
		Array("command", zLogCmd).
		Msg(fmt.Sprintf("running command: %s", cmd.String()))
}

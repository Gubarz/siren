package console

import (
	"encoding/base64"
	"strings"
	"testing"
)

// Control frames travel on the same stream as implant output, so a frame that
// does not carry the job's nonce has to be ignored rather than acted on.
func TestControlFramesMustCarryTheJobNonce(t *testing.T) {
	const nonce = "0123456789abcdef"

	frame := func(prefix, frameNonce, payload string) string {
		return prefix + frameNonce + controlFrameNonceSeparator +
			base64.StdEncoding.EncodeToString([]byte(payload)) + controlFrameSuffix
	}
	encoded := func(payload string) string {
		return base64.StdEncoding.EncodeToString([]byte(payload))
	}

	stream := frame(shellOpenFramePrefix, nonce, "/bin/sh") +
		frame(consoleCommandFramePrefix, nonce, "socks5 start") +
		frame(shellOpenFramePrefix, "attacker-nonce", "/bin/evil") +
		frame(consoleCommandFramePrefix, "", "rm -rf /") +
		// No separator at all, which is what an implant echoing the prefix emits.
		shellOpenFramePrefix + encoded("/bin/evil2") + controlFrameSuffix

	_, shellTails, commands, _ := filterConsoleControlFrames(nonce, nil, []byte(stream))

	if len(shellTails) != 1 || shellTails[0] != "/bin/sh" {
		t.Fatalf("shellTails = %v, want only the frame carrying the job nonce", shellTails)
	}
	if len(commands) != 1 || commands[0] != "socks5 start" {
		t.Fatalf("commands = %v, want only the frame carrying the job nonce", commands)
	}

	// The subprocess side has to put the nonce in the frame it writes.
	controlFrameNonce = nonce
	defer func() { controlFrameNonce = "" }()
	if got := controlFrame(shellOpenFramePrefix, "/bin/sh"); !strings.HasPrefix(got, shellOpenFramePrefix+nonce+controlFrameNonceSeparator) {
		t.Fatalf("controlFrame() = %q, want the job nonce in front of the payload", got)
	}
}

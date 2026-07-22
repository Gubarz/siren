package console

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	commandflags "github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/processes"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/pflag"

	"sliver-gui/internal/sliver/rpc"
)

func (s *Service) runDirectCommand(line string, sess *clientpb.Session, beacon *clientpb.Beacon) (string, bool, error) {
	args, err := shellquote.Split(line)
	if err != nil || len(args) == 0 {
		return "", false, nil
	}

	switch strings.ToLower(args[0]) {
	case "ping":
		return s.runDirectPing(args[1:], sess, beacon)
	case "ps":
		return s.runDirectPS(args[1:], sess, beacon)
	default:
		return "", false, nil
	}
}

func (s *Service) runDirectPing(args []string, sess *clientpb.Session, beacon *clientpb.Beacon) (string, bool, error) {
	flags := newDirectFlagSet("ping")
	timeoutSeconds := flags.Int64P("timeout", "t", commandflags.DefaultTimeout, "grpc timeout in seconds")
	if err := flags.Parse(args); err != nil {
		return "", true, err
	}
	if flags.NArg() > 0 {
		return "", true, fmt.Errorf("ping does not accept arguments")
	}
	if sess == nil {
		return "", true, fmt.Errorf("ping requires an active session")
	}

	nonce := int32(util.Intn(999999))
	out, err := s.output.capture(func() error {
		s.sliverCon.PrintInfof("Ping %d\n", nonce)

		ctx, cancel := context.WithTimeout(context.Background(), rpcContextTimeout(*timeoutSeconds))
		defer cancel()

		pong, err := s.rpc.RPC.Ping(ctx, &sliverpb.Ping{
			Nonce:   nonce,
			Request: targetRequest(sess, beacon, *timeoutSeconds),
		})
		if err != nil {
			s.sliverCon.PrintErrorf("%s\n", err)
			return nil
		}
		if err := rpc.CheckResponse(pong); err != nil {
			s.sliverCon.PrintErrorf("%s\n", err)
			return nil
		}

		s.sliverCon.PrintInfof("Pong %d\n", pong.Nonce)
		return nil
	})
	return strings.TrimRight(out, "\n"), true, err
}

func (s *Service) runDirectPS(args []string, sess *clientpb.Session, beacon *clientpb.Beacon) (string, bool, error) {
	flags := newPSFlagSet()
	if err := flags.Parse(args); err != nil {
		return "", true, err
	}
	if flags.NArg() > 0 {
		return "", true, fmt.Errorf("ps does not accept positional arguments")
	}

	timeoutSeconds, _ := flags.GetInt64("timeout")

	out, err := s.output.capture(func() error {
		fullInfo, proceed := s.validatePSFlags(flags)
		if !proceed {
			return nil
		}
		return s.printProcessList(sess, beacon, timeoutSeconds, fullInfo, flags)
	})
	return strings.TrimRight(out, "\n"), true, err
}

// validatePSFlags reconciles incompatible flag combinations and reports
// misuse. It returns the effective full-info value and whether the command
// should proceed.
func (s *Service) validatePSFlags(flags *pflag.FlagSet) (bool, bool) {
	fullInfo, _ := flags.GetBool("full")
	tree, _ := flags.GetBool("tree")
	ownerFilter, _ := flags.GetString("owner")
	cmdLine, _ := flags.GetBool("print-cmdline")

	if tree && fullInfo {
		s.sliverCon.PrintWarnf("Process tree and full process metadata were requested. Full process metadata is not necessary for the process tree, so the request for full process metadata will be ignored.\n\n")
		fullInfo = false
		_ = flags.Set("full", "false")
	}
	if ownerFilter != "" && !fullInfo {
		s.sliverCon.PrintErrorf("Filtering on process owner was requested, but full process metadata was not requested. Re-run the command, and specify the -f flag.\n")
		return false, false
	}
	if cmdLine && !fullInfo {
		s.sliverCon.PrintErrorf("Process command line arguments were requested, but full process metadata was not requested. Re-run the command, and specify the -f flag.\n")
		return false, false
	}
	return fullInfo, true
}

func (s *Service) printProcessList(sess *clientpb.Session, beacon *clientpb.Beacon, timeoutSeconds int64, fullInfo bool, flags *pflag.FlagSet) error {
	ctx, cancel := context.WithTimeout(context.Background(), rpcContextTimeout(timeoutSeconds))
	defer cancel()

	ps, err := s.rpc.RPC.Ps(ctx, &sliverpb.PsReq{
		FullInfo: fullInfo,
		Request:  targetRequest(sess, beacon, timeoutSeconds),
	})
	if err != nil {
		s.sliverCon.PrintErrorf("%s\n", err)
		return nil
	}
	if err := rpc.CheckResponse(ps); err != nil {
		s.sliverCon.PrintErrorf("%s\n", err)
		return nil
	}
	if ps.Response != nil && ps.Response.Async {
		s.sliverCon.PrintAsyncResponse(ps.Response)
		return nil
	}

	processes.PrintPS(targetOS(sess, beacon), ps, false, fullInfo, flags, s.sliverCon)
	return nil
}

func newDirectFlagSet(name string) *pflag.FlagSet {
	flags := pflag.NewFlagSet(name, pflag.ContinueOnError)
	flags.SetOutput(io.Discard)
	return flags
}

func newPSFlagSet() *pflag.FlagSet {
	flags := newDirectFlagSet("ps")
	flags.IntP("pid", "p", -1, "filter based on pid")
	flags.StringP("exe", "e", "", "filter based on executable name")
	flags.StringP("owner", "o", "", "filter based on owner, must request full process metadata (-f)")
	flags.BoolP("print-cmdline", "c", false, "print command line arguments, must request full process metadata (-f)")
	flags.BoolP("overflow", "O", false, "overflow terminal width (display truncated rows)")
	flags.IntP("skip-pages", "S", 0, "skip the first n page(s)")
	flags.BoolP("tree", "T", false, "print process tree")
	flags.BoolP("full", "f", false, "show full process metadata (owner, architecture, session information) -- may trigger EDR")
	flags.Int64P("timeout", "t", commandflags.DefaultTimeout, "grpc timeout in seconds")
	return flags
}

func targetRequest(sess *clientpb.Session, beacon *clientpb.Beacon, timeoutSeconds int64) *commonpb.Request {
	if timeoutSeconds <= 0 {
		timeoutSeconds = commandflags.DefaultTimeout
	}

	req := &commonpb.Request{Timeout: int64(time.Duration(timeoutSeconds)*time.Second - time.Nanosecond)}
	if sess != nil {
		req.SessionID = sess.ID
		return req
	}
	if beacon != nil {
		req.Async = true
		req.BeaconID = beacon.ID
	}
	return req
}

func rpcContextTimeout(timeoutSeconds int64) time.Duration {
	if timeoutSeconds <= 0 {
		timeoutSeconds = commandflags.DefaultTimeout
	}
	return time.Duration(timeoutSeconds+1) * time.Second
}

func targetOS(sess *clientpb.Session, beacon *clientpb.Beacon) string {
	if sess != nil {
		return sess.OS
	}
	if beacon != nil {
		return beacon.OS
	}
	return ""
}

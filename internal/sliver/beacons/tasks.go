package beacons

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	sliverTasks "github.com/bishopfox/sliver/client/command/tasks"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"

	automationevents "siren/internal/automation/events"
	"siren/internal/bus"
	"siren/internal/execctx"
	"siren/internal/sliver/console"
	"siren/internal/sliver/rpc"
	"siren/internal/sliver/rpcwrap"
)

const beaconTaskPollInterval = time.Second

type Service struct {
	rpc     *rpc.Client
	console *console.Service
	bus     bus.Bus
}

func New(rpc *rpc.Client, con *console.Service) *Service {
	return &Service{rpc: rpc, console: con}
}

func (s *Service) SetBus(b bus.Bus) {
	s.bus = b
}

func (s *Service) GetBeaconTasks(beaconID string) (*clientpb.BeaconTasks, error) {
	return rpcwrap.CallRequired(s.rpc, rpcpb.SliverRPCClient.GetBeaconTasks, &clientpb.Beacon{ID: beaconID}, beaconID, "beacon ID")
}

type TaskOutput struct {
	Type           string                     `json:"type"`
	TextOutput     string                     `json:"textOutput"`
	ImageData      string                     `json:"imageData"`
	Processes      []*commonpb.Process        `json:"processes"`
	NetstatEntries []*sliverpb.SockTabEntry   `json:"netstatEntries"`
	Files          []*sliverpb.FileInfo       `json:"files"`
	Path           string                     `json:"path"`
	Services       []*sliverpb.ServiceDetails `json:"services"`
	EnvVars        []*commonpb.EnvVar         `json:"envVars"`
}

func (s *Service) GetBeaconTaskOutput(taskID string) (*TaskOutput, error) {
	c, err := rpcwrap.Client(s.rpc)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(taskID) == "" {
		return nil, fmt.Errorf("task ID is required")
	}

	task, err := c.RPC.GetBeaconTaskContent(
		context.Background(),
		&clientpb.BeaconTask{ID: taskID},
	)
	if err != nil {
		return nil, err
	}
	return s.decodeTaskOutput(task)
}

// taskOutputCase maps a beacon task description prefix (the protobuf request
// type name, lowercased) to a decoder producing a typed TaskOutput. Adding
// support for a new task kind is a single table entry — decoders never
// re-task the beacon; they decode the stored response.
type taskOutputCase struct {
	prefix string
	decode func(*clientpb.BeaconTask) (*TaskOutput, error)
}

func (s *Service) taskOutputCases() []taskOutputCase {
	return []taskOutputCase{
		{"screenshotreq", typed(func(r *sliverpb.Screenshot) *TaskOutput {
			return &TaskOutput{Type: "image", ImageData: base64.StdEncoding.EncodeToString(r.Data)}
		})},
		{"psreq", typed(func(r *sliverpb.Ps) *TaskOutput {
			return &TaskOutput{Type: "processes", Processes: r.Processes}
		})},
		{"netstatreq", s.withRendered(typed(func(r *sliverpb.Netstat) *TaskOutput {
			return &TaskOutput{Type: "netstat", NetstatEntries: r.Entries}
		}))},
		{"ls", typed(func(r *sliverpb.Ls) *TaskOutput {
			return &TaskOutput{Type: "filelist", Files: r.Files, Path: r.Path}
		})},
		{"services", typed(func(r *sliverpb.Services) *TaskOutput {
			return &TaskOutput{Type: "services", Services: r.Details}
		})},
		{"envreq", s.withRendered(typed(func(r *sliverpb.EnvInfo) *TaskOutput {
			return &TaskOutput{Type: "env", EnvVars: r.Variables}
		}))},
	}
}

func (s *Service) decodeTaskOutput(task *clientpb.BeaconTask) (*TaskOutput, error) {
	desc := strings.ToLower(task.Description)
	for _, c := range s.taskOutputCases() {
		if !strings.HasPrefix(desc, c.prefix) {
			continue
		}
		out, err := c.decode(task)
		if err != nil {
			return s.taskTextFallback(task, err)
		}
		return out, nil
	}
	rendered, err := s.renderBeaconTask(task)
	return &TaskOutput{Type: "text", TextOutput: rendered}, err
}

// typed builds a task decoder that unmarshals the stored response into the
// request's protobuf response type and wraps it as a TaskOutput.
func typed[M any, PM interface {
	*M
	proto.Message
}](wrap func(PM) *TaskOutput) func(*clientpb.BeaconTask) (*TaskOutput, error) {
	return func(task *clientpb.BeaconTask) (*TaskOutput, error) {
		var msg M
		pm := PM(&msg)
		if err := proto.Unmarshal(task.Response, pm); err != nil {
			return nil, err
		}
		return wrap(pm), nil
	}
}

// withRendered attaches the rendered task text alongside the typed payload
// so the inline task view still has something to show.
func (s *Service) withRendered(
	decode func(*clientpb.BeaconTask) (*TaskOutput, error),
) func(*clientpb.BeaconTask) (*TaskOutput, error) {
	return func(task *clientpb.BeaconTask) (*TaskOutput, error) {
		out, err := decode(task)
		if err != nil {
			return nil, err
		}
		rendered, _ := s.renderBeaconTask(task)
		out.TextOutput = rendered
		return out, nil
	}
}

func (s *Service) taskTextFallback(task *clientpb.BeaconTask, cause error) (*TaskOutput, error) {
	log.Printf("beacons: failed to unmarshal %s response: %v", strings.ToLower(task.Description), cause)
	rendered, rerr := s.renderBeaconTask(task)
	return &TaskOutput{Type: "text", TextOutput: rendered}, rerr
}

func (s *Service) CancelBeaconTask(taskID string) error {
	_, err := rpcwrap.CallRequired(s.rpc, rpcpb.SliverRPCClient.CancelBeaconTask, &clientpb.BeaconTask{ID: taskID}, taskID, "task ID")
	return err
}

func (s *Service) AwaitBeaconTask(
	ctx context.Context,
	beaconID string,
	commandOutput string,
	taskID string,
) (string, bool, error) {
	matches := console.BeaconTaskNoticePattern.FindStringSubmatch(commandOutput)
	if len(matches) != 2 {
		return commandOutput, false, nil
	}

	task, err := s.awaitTask(ctx, beaconID, matches[1], taskID)
	if err != nil {
		return commandOutput, true, err
	}

	state := strings.ToLower(task.State)
	if state != "completed" {
		if state == "" {
			state = "unknown"
		}
		err := fmt.Errorf("beacon task %s %s", shortTaskID(task.ID), state)
		s.publishBeaconTaskResult(ctx, beaconID, err)
		return commandOutput, true, err
	}

	rendered, err := s.renderBeaconTask(task)
	if err != nil {
		s.publishBeaconTaskResult(ctx, beaconID, err)
		return commandOutput, true, err
	}
	s.publishBeaconTaskResult(ctx, beaconID, nil)
	return rendered, true, nil
}

func (s *Service) awaitTask(ctx context.Context, beaconID, prefix, taskID string) (*clientpb.BeaconTask, error) {
	var err error
	if taskID == "" {
		taskID, err = s.resolveBeaconTaskID(ctx, beaconID, prefix)
	}
	if err != nil {
		return nil, err
	}
	s.console.RemoveBeaconTaskCallback(taskID)

	task, err := s.waitForBeaconTask(ctx, taskID)
	if err != nil {
		if shouldCancelPendingBeaconTask(err) {
			s.cancelPendingBeaconTask(taskID)
		}
		err = fmt.Errorf("beacon task %s: %w", shortTaskID(taskID), err)
		s.publishBeaconTaskResult(ctx, beaconID, err)
		return nil, err
	}
	return task, nil
}

func (s *Service) resolveBeaconTaskID(ctx context.Context, beaconID, prefix string) (string, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		tasks, err := s.rpc.RPC.GetBeaconTasks(ctx, &clientpb.Beacon{ID: beaconID})
		if err != nil {
			return "", err
		}
		for _, task := range tasks.Tasks {
			if strings.HasPrefix(strings.ToLower(task.ID), strings.ToLower(prefix)) {
				return task.ID, nil
			}
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *Service) waitForBeaconTask(ctx context.Context, taskID string) (*clientpb.BeaconTask, error) {
	ticker := time.NewTicker(beaconTaskPollInterval)
	defer ticker.Stop()

	for {
		task, err := s.rpc.RPC.GetBeaconTaskContent(ctx, &clientpb.BeaconTask{ID: taskID})
		if err != nil {
			return nil, err
		}
		switch strings.ToLower(task.State) {
		case "completed", "failed", "canceled":
			return task, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *Service) renderBeaconTask(task *clientpb.BeaconTask) (string, error) {
	if s.console == nil {
		return "", nil
	}
	return s.console.Render(func() error {
		sliverTasks.PrintTask(task, s.console.SliverCon())
		return nil
	})
}

func (s *Service) cancelPendingBeaconTask(taskID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	task, err := s.rpc.RPC.GetBeaconTaskContent(ctx, &clientpb.BeaconTask{ID: taskID})
	if err != nil || strings.ToLower(task.State) != "pending" {
		return
	}
	_, _ = s.rpc.RPC.CancelBeaconTask(ctx, &clientpb.BeaconTask{ID: taskID})
}

func shouldCancelPendingBeaconTask(err error) bool {
	return err != nil && !errors.Is(err, context.Canceled)
}

func shortTaskID(taskID string) string {
	if index := strings.IndexByte(taskID, '-'); index >= 0 {
		return taskID[:index]
	}
	return taskID
}

func (s *Service) publishBeaconTaskResult(ctx context.Context, beaconID string, taskErr error) {
	if s.bus == nil {
		return
	}
	_, _, hostname := execctx.Target(ctx)
	status := "ok"
	errText := ""
	if taskErr != nil {
		status = "error"
		errText = taskErr.Error()
	}
	s.bus.Publish(bus.Event{
		Type:   "beacon.task-result",
		Source: "gui",
		Payload: automationevents.TaskResult{
			Verb:       "BeaconTaskResult",
			TargetID:   beaconID,
			TargetKind: "beacon",
			Hostname:   hostname,
			Status:     status,
			Error:      errText,
		},
	})
}

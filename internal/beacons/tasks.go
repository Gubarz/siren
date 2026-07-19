package beacons

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	sliverTasks "github.com/bishopfox/sliver/client/command/tasks"
	"github.com/bishopfox/sliver/protobuf/clientpb"

	"sliver-gui/internal/console"
	"sliver-gui/internal/rpc"
)

const beaconTaskPollInterval = time.Second

type Service struct {
	rpc     *rpc.Client
	console *console.Service
}

func New(rpc *rpc.Client, con *console.Service) *Service {
	return &Service{rpc: rpc, console: con}
}

func (s *Service) GetBeaconTasks(beaconID string) (*clientpb.BeaconTasks, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	if strings.TrimSpace(beaconID) == "" {
		return nil, fmt.Errorf("beacon ID is required")
	}
	return s.rpc.RPC.GetBeaconTasks(
		context.Background(),
		&clientpb.Beacon{ID: beaconID},
	)
}

func (s *Service) GetBeaconTaskOutput(taskID string) (string, error) {
	if !s.rpc.Connected() {
		return "", rpc.ErrNotConnected
	}
	if strings.TrimSpace(taskID) == "" {
		return "", fmt.Errorf("task ID is required")
	}

	task, err := s.rpc.RPC.GetBeaconTaskContent(
		context.Background(),
		&clientpb.BeaconTask{ID: taskID},
	)
	if err != nil {
		return "", err
	}

	return s.renderBeaconTask(task)
}

func (s *Service) CancelBeaconTask(taskID string) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	if strings.TrimSpace(taskID) == "" {
		return fmt.Errorf("task ID is required")
	}
	_, err := s.rpc.RPC.CancelBeaconTask(
		context.Background(),
		&clientpb.BeaconTask{ID: taskID},
	)
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

	prefix := matches[1]
	var err error
	if taskID == "" {
		taskID, err = s.resolveBeaconTaskID(ctx, beaconID, prefix)
	}
	if err != nil {
		return commandOutput, true, err
	}
	s.console.RemoveBeaconTaskCallback(taskID)

	task, err := s.waitForBeaconTask(ctx, taskID)
	if err != nil {
		if shouldCancelPendingBeaconTask(err) {
			s.cancelPendingBeaconTask(taskID)
		}
		return commandOutput, true, fmt.Errorf("beacon task %s: %w", shortTaskID(taskID), err)
	}

	state := strings.ToLower(task.State)
	if state != "completed" {
		if state == "" {
			state = "unknown"
		}
		return commandOutput, true, fmt.Errorf("beacon task %s %s", shortTaskID(task.ID), state)
	}

	rendered, err := s.renderBeaconTask(task)
	if err != nil {
		return commandOutput, true, err
	}
	return rendered, true, nil
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

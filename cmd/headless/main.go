package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"sliver-gui/internal/bootstrap"
	"sliver-gui/internal/envvars"
)

type headlessEmitter struct{}

func (headlessEmitter) Emit(name string, payload any) {
	data, err := json.Marshal(map[string]any{"event": name, "payload": payload})
	if err != nil {
		return
	}
	fmt.Println(string(data))
}

func main() {
	profile := flag.String("profile", "", "sliver client profile name")
	flag.Parse()

	dataDir, err := envvars.ResolveDataDir(nil)
	if err != nil {
		dataDir = filepath.Join(os.TempDir(), fmt.Sprintf("sliver-gui-%d", os.Getpid()))
		_ = os.MkdirAll(dataDir, 0o700)
	}

	emitter := headlessEmitter{}
	shared := bootstrap.NewShared(bootstrap.Dependencies{
		Emitter: emitter,
		DataDir: dataDir,
	})

	if err := shared.RPC.Connect(*profile); err != nil {
		log.Fatalf("connect: %v", err)
	}

	cfg := shared.RPC.Config
	if cfg != nil {
		shared.Automation.SetServer(cfg.LHost, uint32(cfg.LPort))
		shared.Tags.SetServer(cfg.LHost, uint32(cfg.LPort))
		shared.Comments.SetServer(cfg.LHost, uint32(cfg.LPort))
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	shared.Console.SetEmitter(emitter)
	shared.Automation.Start(ctx)
	shared.CheckinPub.Start(ctx)

	log.Printf("headless: connected to %s:%d as %s", cfg.LHost, cfg.LPort, cfg.Operator)
	<-ctx.Done()
	log.Println("headless: shutting down")
	if err := shared.Journal.Close(); err != nil {
		log.Printf("headless: close journal: %v", err)
	}
	shared.RPC.Disconnect()
}

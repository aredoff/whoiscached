package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/whoiscache/whoiscache/internal/cache"
	"github.com/whoiscache/whoiscache/internal/config"
	"github.com/whoiscache/whoiscache/internal/server"
	"github.com/whoiscache/whoiscache/internal/service"
	"github.com/whoiscache/whoiscache/internal/version"
	"github.com/whoiscache/whoiscache/internal/whois"
)

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "version" {
		fmt.Println(version.String())
		return
	}
	showVersion := flag.Bool("version", false, "print version and exit")
	cfgPath := flag.String("config", "configs/config.ini", "path to INI")
	dumpKeys := flag.Bool("dump-keys", false, "list RecordKeys from snapshot file and exit")
	getKey := flag.String("get-key", "", "print primary body for RecordKey from snapshot and exit")
	deleteKey := flag.String("delete-key", "", "remove RecordKey from store and snapshot, then exit")
	flag.Parse()
	if *showVersion {
		fmt.Println(version.String())
		return
	}

	cfg, err := config.LoadFromEnv(*cfgPath)
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}

	if *deleteKey != "" {
		if err := runDeleteCLI(cfg, *deleteKey); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	if *dumpKeys || *getKey != "" {
		if err := runSnapshotCLI(cfg, *dumpKeys, *getKey); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	store, err := cache.NewDiskStore(cfg.Storage.SnapshotPath, cfg.Storage.SnapshotInterval)
	if err != nil {
		slog.Error("disk store", "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := store.Close(); err != nil {
			slog.Error("store close", "err", err)
		}
	}()

	wclient := &whois.Client{MaxResponseBytes: cfg.Whois.MaxResponseBytes}
	res := &whois.Resolver{
		Client:   wclient,
		RootIANA: whois.HostPort(cfg.Whois.DomainRootServer),
		MaxHops:  cfg.Whois.MaxReferralHops,
		Timeout:  cfg.Whois.DefaultTimeout,
		Conf:     &cfg.Whois,
	}
	svc := service.New(cfg, res, store)
	srv := server.NewTCPServer(cfg, svc)

	sctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err = srv.Serve(sctx); err != nil {
		slog.Error("serve", "err", err)
		os.Exit(1)
	}
}

func runSnapshotCLI(cfg *config.Config, dumpKeys bool, getKey string) error {
	rows, err := cache.ReadSnapshotFile(cfg.Storage.SnapshotPath)
	if err != nil {
		return err
	}
	if dumpKeys {
		for _, k := range cache.SortedSnapKeys(rows) {
			fmt.Println(k)
		}
		return nil
	}
	if b, ok := cache.SnapGetPrimary(rows, getKey); ok {
		if _, err := os.Stdout.Write(b); err != nil {
			return err
		}
		if len(b) > 0 && b[len(b)-1] != '\n' {
			fmt.Println()
		}
		return nil
	}
	return fmt.Errorf("key not found or primary expired: %q", getKey)
}

func runDeleteCLI(cfg *config.Config, key string) error {
	store, err := cache.NewDiskStore(cfg.Storage.SnapshotPath, cfg.Storage.SnapshotInterval)
	if err != nil {
		return err
	}
	defer func() {
		if err := store.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "store close:", err)
		}
	}()
	if err := store.Delete(context.Background(), key); err != nil {
		if errors.Is(err, cache.ErrNotFound) {
			return fmt.Errorf("no such key: %q", key)
		}
		return err
	}
	return nil
}

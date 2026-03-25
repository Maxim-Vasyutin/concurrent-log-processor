package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cli_tool/internal/cli"
	"cli_tool/internal/processor"
	"cli_tool/internal/reporter"
	"cli_tool/internal/scanner"
)

func main() {
	if err := run(os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(out io.Writer) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signalCh)

	go func() {
		select {
		case <-signalCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	cfg, err := cli.ParseCommandLineArgs()
	if err != nil {
		return err
	}
	scanResult, err := scanner.LogFileScanner{}.Scan(cfg.InputDir)
	if err != nil {
		return err
	}
	started := time.Now()
	processResult := processor.New().ProcessFilesWithContext(ctx, scanResult.LogFiles)
	elapsed := time.Since(started)
	if err := reporter.WriteJSONReport(processResult.Analysis, cfg.OutputFile); err != nil {
		return err
	}

	_, err = fmt.Fprintf(
		out,
		"discovered_files=%d scan_errors=%d processed_files=%d failed_files=%d total_lines=%d valid_entries=%d parse_errors=%d merged_entries=%d request_groups=%d orphaned_records=%d failed_requests=%d processing_ms=%d interrupted=%t\n",
		len(scanResult.LogFiles),
		len(scanResult.ScanErrors),
		processResult.ProcessedFiles,
		processResult.FailedFiles,
		processResult.TotalLines,
		processResult.ValidEntries,
		processResult.ParseErrors,
		processResult.MergedEntries,
		processResult.RequestGroups,
		processResult.OrphanedRecords,
		processResult.FailedRequests,
		elapsed.Milliseconds(),
		ctx.Err() != nil,
	)
	return err
}

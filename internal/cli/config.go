package cli

import (
	"flag"
	"fmt"
	"os"
)

type Config struct {
	InputDir   string
	OutputFile string
}

func ParseCommandLineArgs() (Config, error) {
	flagSet := flag.NewFlagSet("cli_tool", flag.ContinueOnError)

	var cfg Config
	flagSet.StringVar(&cfg.InputDir, "input-dir", ".", "input directory with log files")
	flagSet.StringVar(&cfg.OutputFile, "output-file", "results.json", "output report filename")

	if err := flagSet.Parse(os.Args[1:]); err != nil {
		return Config{}, fmt.Errorf("parse command line args: %w", err)
	}
	if flagSet.NArg() != 0 {
		return Config{}, fmt.Errorf("unexpected positional arguments")
	}

	info, err := os.Stat(cfg.InputDir)
	if err != nil {
		return Config{}, fmt.Errorf("input directory %q: %w", cfg.InputDir, err)
	}
	if !info.IsDir() {
		return Config{}, fmt.Errorf("input path %q is not a directory", cfg.InputDir)
	}

	return cfg, nil
}

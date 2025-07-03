// cli.go: Command-line interface for TmuxAI, including root command and flags

package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/alvinunreal/tmuxai/config"
	"github.com/alvinunreal/tmuxai/internal"
	"github.com/alvinunreal/tmuxai/logger"
	"github.com/spf13/cobra"
)

var (
	initMessage  string
	taskFileFlag string
	agenticFlag  bool
	layoutFlag   string
)

var rootCmd = &cobra.Command{
	Use:   "tmuxai [request message]",
	Short: "TmuxAI - AI-Powered Tmux Companion",
	Long:  `TmuxAI - AI-Powered Tmux Companion`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if v, _ := cmd.Flags().GetBool("version"); v {
			fmt.Printf("tmuxai version: %s\ncommit: %s\nbuild date: %s\n", internal.Version, internal.Commit, internal.Date)
			os.Exit(0)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load()
		if err != nil {
			logger.Error("Error loading configuration: %v", err)
			fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
			os.Exit(1)
		}

		if agenticFlag {
			cfg.AgenticMode = true
		}

		if len(args) > 0 {
			initMessage = strings.Join(args, " ")
		}

		if taskFileFlag != "" {
			content, err := os.ReadFile(taskFileFlag)
			if err != nil {
				logger.Error("Error reading task file: %v", err)
				fmt.Fprintf(os.Stderr, "Error reading task file: %v\n", err)
				os.Exit(1)
			}
			initMessage = string(content)
			logger.Info("Read request from file: %s", taskFileFlag)
		}

		mgr, err := internal.NewManager(cfg)
		if err != nil {
			logger.Error("manager.NewManager failed: %v", err)
			os.Exit(1)
		}

		// Handle layout - either from flag or config
		layout := layoutFlag
		if layout == "" && cfg.Layout != "" {
			layout = cfg.Layout
		}
		
		if layout != "" {
			if !mgr.GetAgenticMode() {
				logger.Error("Layout requires agentic mode to be enabled.")
				fmt.Fprintln(os.Stderr, "Error: Layout requires agentic mode to be enabled. Use --agentic flag or set agentic_mode: true in config.")
				os.Exit(1)
			}
			if err := mgr.ApplyLayout(layout); err != nil {
				logger.Error("Failed to apply layout: %v", err)
				fmt.Fprintf(os.Stderr, "Error applying layout: %v\n", err)
				os.Exit(1)
			}
		}
		if initMessage != "" {
			logger.Info("Starting with initial subcommand: %s", initMessage)
		}

		if err := mgr.Start(initMessage); err != nil {
			logger.Error("manager.Start failed: %v", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.Flags().StringVarP(&taskFileFlag, "file", "f", "", "Read request from specified file")
	rootCmd.Flags().BoolP("version", "v", false, "Print version information")
	rootCmd.Flags().BoolVar(&agenticFlag, "agentic", false, "Enable agentic multi-pane features")
	rootCmd.Flags().StringVar(&layoutFlag, "layout", "", "Specify a layout to apply on startup (e.g., '1x2'). Requires --agentic flag.")
}

func Execute() error {
	return rootCmd.Execute()
}

package cli

import (
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	"github.com/Omotolani98/rekord/internal/agent"
	"github.com/Omotolani98/rekord/internal/config"
	"github.com/Omotolani98/rekord/internal/live"
)

func newMcpCommand() *cobra.Command {
	var root, cfgPath string
	var noRedact bool

	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Run an MCP server for live agent-driven terminal sessions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runMcp(cmd, root, cfgPath, noRedact)
		},
	}

	cmd.Flags().StringVar(&root, "root", defaultSessionsRoot(), "sessions root directory")
	cmd.Flags().StringVar(&cfgPath, "config", defaultConfigPath(), "config file with redaction patterns")
	cmd.Flags().BoolVar(&noRedact, "no-redact", false, "disable redaction of captures and logs")

	return cmd
}

func runMcp(cmd *cobra.Command, root, cfgPath string, noRedact bool) error {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}
	redactor, err := buildRedactor(cfg)
	if err != nil {
		return err
	}
	if noRedact {
		redactor = nil
	}

	hub := live.NewHub(root, Version())
	defer hub.Shutdown()

	srv := agent.NewServer(hub, redactor, Version())

	if err := srv.Run(cmd.Context(), &mcp.StdioTransport{}); err != nil {
		return fmt.Errorf("mcp server: %w", err)
	}
	return nil
}

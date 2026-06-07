package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/Omotolani98/rekord/internal/live"
	"github.com/Omotolani98/rekord/internal/session"
)

func newSessionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Manage persistent named terminal sessions over a local socket",
	}
	cmd.AddCommand(
		newSessionStartCommand(),
		newSessionServeCommand(),
		newSessionSendCommand(),
		newSessionShowCommand(),
		newSessionWaitCommand(),
		newSessionStatusCommand(),
		newSessionListCommand(),
		newSessionStopCommand(),
	)
	return cmd
}

func sessionSocketPath(root, name string) string {
	return filepath.Join(root, name+".sock")
}

func sessionLogPath(root, name string) string {
	return filepath.Join(root, name+".session.log")
}

func sessionDo(root, name string, req live.Request) (live.Response, error) {
	if err := session.ValidateName(name); err != nil {
		return live.Response{}, fmt.Errorf("--name is required: %w", err)
	}
	sock := sessionSocketPath(root, name)
	resp, err := live.Do(sock, req)
	if err != nil && resp.Error == "" {
		return live.Response{}, fmt.Errorf("session %q not running", name)
	}
	return resp, err
}

func newSessionStartCommand() *cobra.Command {
	var name, root, cwd string
	var cols, rows int

	cmd := &cobra.Command{
		Use:   "start --name <name> -- <command> [args...]",
		Short: "Launch a persistent session as a detached background process",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSessionStart(cmd, name, root, cwd, cols, rows, args)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "session name (required)")
	cmd.Flags().StringVar(&root, "root", defaultSessionsRoot(), "sessions root directory")
	cmd.Flags().StringVar(&cwd, "cwd", "", "working directory for the command")
	cmd.Flags().IntVar(&cols, "cols", 0, "terminal columns (default 80)")
	cmd.Flags().IntVar(&rows, "rows", 0, "terminal rows (default 24)")
	return cmd
}

func runSessionStart(cmd *cobra.Command, name, root, cwd string, cols, rows int, command []string) error {
	if err := session.ValidateName(name); err != nil {
		return fmt.Errorf("--name is required: %w", err)
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return fmt.Errorf("create sessions root: %w", err)
	}

	sock := sessionSocketPath(root, name)
	if live.Ping(sock) {
		return fmt.Errorf("session %q already running", name)
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}

	serveArgs := []string{"session", "serve", "--name", name, "--root", root, "--socket", sock}
	if cols > 0 {
		serveArgs = append(serveArgs, "--cols", strconv.Itoa(cols))
	}
	if rows > 0 {
		serveArgs = append(serveArgs, "--rows", strconv.Itoa(rows))
	}
	if cwd != "" {
		serveArgs = append(serveArgs, "--cwd", cwd)
	}
	serveArgs = append(serveArgs, "--")
	serveArgs = append(serveArgs, command...)

	logFile, err := os.OpenFile(sessionLogPath(root, name), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open session log: %w", err)
	}
	defer func() { _ = logFile.Close() }()

	proc := exec.Command(exe, serveArgs...)
	proc.SysProcAttr = detachAttr()
	proc.Stdin = nil
	proc.Stdout = logFile
	proc.Stderr = logFile
	if err := proc.Start(); err != nil {
		return fmt.Errorf("start session daemon: %w", err)
	}
	_ = proc.Process.Release()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if live.Ping(sock) {
			fmt.Fprintf(cmd.OutOrStdout(), "started session %q\n", name)
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("session %q did not come up; see %s", name, sessionLogPath(root, name))
}

func newSessionServeCommand() *cobra.Command {
	var name, root, socket, cwd string
	var cols, rows int

	cmd := &cobra.Command{
		Use:    "serve --name <name> --socket <path> -- <command> [args...]",
		Short:  "Run a session daemon in the foreground (internal)",
		Hidden: true,
		Args:   cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSessionServe(cmd, name, root, socket, cwd, cols, rows, args)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "session name")
	cmd.Flags().StringVar(&root, "root", defaultSessionsRoot(), "sessions root directory")
	cmd.Flags().StringVar(&socket, "socket", "", "unix socket path")
	cmd.Flags().StringVar(&cwd, "cwd", "", "working directory")
	cmd.Flags().IntVar(&cols, "cols", 0, "terminal columns")
	cmd.Flags().IntVar(&rows, "rows", 0, "terminal rows")
	return cmd
}

func runSessionServe(cmd *cobra.Command, name, root, socket, cwd string, cols, rows int, command []string) error {
	if socket == "" {
		socket = sessionSocketPath(root, name)
	}
	ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	hub := live.NewHub(root, Version())
	defer hub.Shutdown()

	s, err := hub.Launch(ctx, live.LaunchOptions{
		Name:    name,
		Command: command,
		CWD:     cwd,
		Cols:    cols,
		Rows:    rows,
	})
	if err != nil {
		return fmt.Errorf("launch session: %w", err)
	}

	return live.Serve(ctx, socket, s)
}

func newSessionSendCommand() *cobra.Command {
	var name, root string
	var keys []string

	cmd := &cobra.Command{
		Use:   "send --name <name> [text]",
		Short: "Send text and/or named keys to a session",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := sessionDo(root, name, live.Request{
				Op:   "send",
				Text: strings.Join(args, " "),
				Keys: keys,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "sent %d bytes\n", resp.Sent)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "session name (required)")
	cmd.Flags().StringVar(&root, "root", defaultSessionsRoot(), "sessions root directory")
	cmd.Flags().StringArrayVar(&keys, "key", nil, "named key to send (e.g. enter, ctrl-c); repeatable")
	return cmd
}

func newSessionShowCommand() *cobra.Command {
	var name, root, format string

	cmd := &cobra.Command{
		Use:   "show --name <name>",
		Short: "Show the current screen frame of a session",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			resp, err := sessionDo(root, name, live.Request{Op: "capture"})
			if err != nil {
				return err
			}
			if resp.Frame == nil {
				return fmt.Errorf("no frame returned")
			}
			if format == "json" {
				data, err := json.MarshalIndent(resp.Frame, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), resp.Frame.Text())
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "session name (required)")
	cmd.Flags().StringVar(&root, "root", defaultSessionsRoot(), "sessions root directory")
	cmd.Flags().StringVar(&format, "format", "text", "output format: text or json")
	return cmd
}

func newSessionWaitCommand() *cobra.Command {
	var name, root, text string
	var idle, timeout time.Duration
	var exit bool

	cmd := &cobra.Command{
		Use:   "wait --name <name>",
		Short: "Wait for text, idle output, or process exit",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			req := live.Request{TimeoutMs: int(timeout.Milliseconds())}
			switch {
			case text != "":
				req.Op = "wait_text"
				req.Sub = text
			case exit:
				req.Op = "wait_exit"
			case idle > 0:
				req.Op = "wait_idle"
				req.QuietMs = int(idle.Milliseconds())
			default:
				return fmt.Errorf("specify --text, --idle, or --exit")
			}
			resp, err := sessionDo(root, name, req)
			if err != nil {
				return err
			}
			if resp.ExitCode != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "%s (exit %d)\n", resp.Reason, *resp.ExitCode)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), resp.Reason)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "session name (required)")
	cmd.Flags().StringVar(&root, "root", defaultSessionsRoot(), "sessions root directory")
	cmd.Flags().StringVar(&text, "text", "", "wait until this text appears")
	cmd.Flags().DurationVar(&idle, "idle", 0, "wait until output is quiet for this long")
	cmd.Flags().BoolVar(&exit, "exit", false, "wait until the process exits")
	cmd.Flags().DurationVar(&timeout, "timeout", 10*time.Second, "overall timeout")
	return cmd
}

func newSessionStatusCommand() *cobra.Command {
	var name, root string
	cmd := &cobra.Command{
		Use:   "status --name <name>",
		Short: "Show session status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			resp, err := sessionDo(root, name, live.Request{Op: "status"})
			if err != nil {
				return err
			}
			printStatus(cmd.OutOrStdout(), resp.Status)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "session name (required)")
	cmd.Flags().StringVar(&root, "root", defaultSessionsRoot(), "sessions root directory")
	return cmd
}

func newSessionListCommand() *cobra.Command {
	var root string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List running sessions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			entries, err := filepath.Glob(filepath.Join(root, "*.sock"))
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "%-20s %-8s %s\n", "NAME", "RUNNING", "SIZE")
			for _, sock := range entries {
				name := strings.TrimSuffix(filepath.Base(sock), ".sock")
				resp, err := live.Do(sock, live.Request{Op: "status"})
				if err != nil || resp.Status == nil {
					fmt.Fprintf(out, "%-20s %-8s %s\n", name, "stale", "-")
					continue
				}
				fmt.Fprintf(out, "%-20s %-8t %dx%d\n", name, resp.Status.Running, resp.Status.Cols, resp.Status.Rows)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&root, "root", defaultSessionsRoot(), "sessions root directory")
	return cmd
}

func newSessionStopCommand() *cobra.Command {
	var name, root string
	cmd := &cobra.Command{
		Use:   "stop --name <name>",
		Short: "Stop a session and finalize its recording",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if _, err := sessionDo(root, name, live.Request{Op: "stop"}); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "stopped session %q\n", name)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "session name (required)")
	cmd.Flags().StringVar(&root, "root", defaultSessionsRoot(), "sessions root directory")
	return cmd
}

func printStatus(w io.Writer, st *live.Status) {
	if st == nil {
		fmt.Fprintln(w, "no status")
		return
	}
	if st.Running {
		fmt.Fprintf(w, "%s: running %dx%d (id %s)\n", st.Name, st.Cols, st.Rows, st.ID)
		return
	}
	code := 0
	if st.ExitCode != nil {
		code = *st.ExitCode
	}
	fmt.Fprintf(w, "%s: exited (code %d, id %s)\n", st.Name, code, st.ID)
}

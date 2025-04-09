package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Constants
const (
	DEFAULT_LIMIT    = 10               // Default number of commands to list
	DEFAULT_SHELL    = "bash"           // Default shell to use
	COMMAND_TIMEOUT  = 30 * time.Second // Default timeout for commands
	MAX_OUTPUT_SIZE  = 1024 * 1024      // 1MB max output size
	MAX_HISTORY_SIZE = 100              // Maximum commands to keep in history
)

// CommandExecution stores information about an executed command
type CommandExecution struct {
	Command     string    `json:"command"`
	Shell       string    `json:"shell"`
	Output      string    `json:"output"`
	ExitCode    int       `json:"exitCode"`
	StartTime   time.Time `json:"startTime"`
	EndTime     time.Time `json:"endTime"`
	ExecutionMs int64     `json:"executionMs"`
}

// ShellServer implements the MCP server for shell command execution
type ShellServer struct {
	allowedCommands  []string
	allowAllCommands bool
	commandHistory   []CommandExecution
	historyMutex     sync.Mutex
	server           *server.MCPServer
}

// NewShellServer creates a new shell server with the given allowed commands
func NewShellServer(allowedCommands string) (*ShellServer, error) {
	var cmdList []string
	allowAll := false

	// Parse allowed commands
	if allowedCommands == "*" {
		allowAll = true
		cmdList = []string{}
	} else {
		// Split by comma and trim spaces
		for _, cmd := range strings.Split(allowedCommands, ",") {
			trimmed := strings.TrimSpace(cmd)
			if trimmed != "" {
				cmdList = append(cmdList, trimmed)
			}
		}
	}

	s := &ShellServer{
		allowedCommands:  cmdList,
		allowAllCommands: allowAll,
		commandHistory:   make([]CommandExecution, 0, MAX_HISTORY_SIZE),
		server: server.NewMCPServer(
			"unix-shell-server",
			"0.1.0",
			server.WithResourceCapabilities(false, false),
		),
	}

	// Register tool handlers
	s.server.AddTool(mcp.NewTool(
		"execute_command",
		mcp.WithDescription("Execute a shell command using bash or zsh."),
		mcp.WithString("command",
			mcp.Description("The command to execute"),
			mcp.Required(),
		),
		mcp.WithString("shell",
			mcp.Description("The shell to use (bash or zsh)"),
		),
	), s.handleExecuteCommand)

	s.server.AddTool(mcp.NewTool(
		"list_recent_commands",
		mcp.WithDescription("List recently executed commands."),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of commands to return"),
		),
	), s.handleListRecentCommands)

	s.server.AddTool(mcp.NewTool(
		"list_allowed_commands",
		mcp.WithDescription("List all commands that are allowed to be executed."),
	), s.handleListAllowedCommands)

	return s, nil
}

// isCommandAllowed checks if a command is in the allowed list
func (s *ShellServer) isCommandAllowed(command string) bool {
	if s.allowAllCommands {
		return true
	}

	// Extract the base command (first word before any spaces)
	baseCmd := strings.Fields(command)
	if len(baseCmd) == 0 {
		return false
	}

	// Check if the base command is in the allowed list
	for _, allowed := range s.allowedCommands {
		if baseCmd[0] == allowed {
			return true
		}
	}

	return false
}

// addToHistory adds a command execution to the history
func (s *ShellServer) addToHistory(execution CommandExecution) {
	s.historyMutex.Lock()
	defer s.historyMutex.Unlock()

	// Add to the front of the list
	s.commandHistory = append([]CommandExecution{execution}, s.commandHistory...)

	// Trim if exceeding max size
	if len(s.commandHistory) > MAX_HISTORY_SIZE {
		s.commandHistory = s.commandHistory[:MAX_HISTORY_SIZE]
	}
}

// getHistory returns the command history (up to limit)
func (s *ShellServer) getHistory(limit int) []CommandExecution {
	s.historyMutex.Lock()
	defer s.historyMutex.Unlock()

	if limit <= 0 || limit > len(s.commandHistory) {
		limit = len(s.commandHistory)
	}

	result := make([]CommandExecution, limit)
	copy(result, s.commandHistory[:limit])
	return result
}

// executeCommand executes a shell command and returns its output
func (s *ShellServer) executeCommand(command string, shell string) CommandExecution {
	if shell == "" {
		shell = DEFAULT_SHELL
	}

	// Only allow bash or zsh
	if shell != "bash" && shell != "zsh" {
		return CommandExecution{
			Command:   command,
			Shell:     shell,
			Output:    fmt.Sprintf("Error: Unsupported shell '%s'. Only bash and zsh are supported.", shell),
			ExitCode:  1,
			StartTime: time.Now(),
			EndTime:   time.Now(),
		}
	}

	execution := CommandExecution{
		Command:   command,
		Shell:     shell,
		StartTime: time.Now(),
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), COMMAND_TIMEOUT)
	defer cancel()

	// Create the command
	cmd := exec.CommandContext(ctx, shell, "-c", command)

	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()

	execution.EndTime = time.Now()
	execution.ExecutionMs = execution.EndTime.Sub(execution.StartTime).Milliseconds()

	// Truncate output if it's too large
	outputStr := string(output)
	if len(outputStr) > MAX_OUTPUT_SIZE {
		outputStr = outputStr[:MAX_OUTPUT_SIZE] + "\n... (output truncated due to size limit)"
	}
	execution.Output = outputStr

	// Handle different error types
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			execution.Output += "\n\nError: Command execution timed out after 30 seconds."
			execution.ExitCode = 124 // Common timeout exit code
		} else if exitError, ok := err.(*exec.ExitError); ok {
			execution.ExitCode = exitError.ExitCode()
		} else {
			execution.Output += "\n\nError: " + err.Error()
			execution.ExitCode = 1
		}
	} else {
		execution.ExitCode = 0
	}

	return execution
}

// Tool handlers
func (s *ShellServer) handleExecuteCommand(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	command, ok := request.Params.Arguments["command"].(string)
	if !ok {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: 'command' must be a string",
				},
			},
			IsError: true,
		}, nil
	}

	// Get optional shell parameter
	shell := DEFAULT_SHELL
	if shellArg, ok := request.Params.Arguments["shell"].(string); ok && shellArg != "" {
		shell = shellArg
	}

	// Check if command is allowed
	if !s.isCommandAllowed(command) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf(
						"Error: Command '%s' is not in the allowed list. Run 'list_allowed_commands' to see what commands are permitted.",
						strings.Fields(command)[0],
					),
				},
			},
			IsError: true,
		}, nil
	}

	// Execute the command
	execution := s.executeCommand(command, shell)

	// Add to history
	s.addToHistory(execution)

	// Construct the response
	var executionStatus string
	if execution.ExitCode == 0 {
		executionStatus = "completed successfully"
	} else {
		executionStatus = fmt.Sprintf("failed with exit code %d", execution.ExitCode)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf(
					"$ %s\n\n%s\n\nCommand %s in %d ms",
					command,
					execution.Output,
					executionStatus,
					execution.ExecutionMs,
				),
			},
		},
	}, nil
}

func (s *ShellServer) handleListRecentCommands(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	// Get optional limit parameter
	limit := DEFAULT_LIMIT
	if limitArg, ok := request.Params.Arguments["limit"].(float64); ok {
		limit = int(limitArg)
	}

	// Get command history
	history := s.getHistory(limit)

	if len(history) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "No commands have been executed yet.",
				},
			},
		}, nil
	}

	// Format the response
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Recent commands (showing %d of %d total):\n\n",
		len(history), len(s.commandHistory)))

	for i, cmd := range history {
		statusMsg := "Success"
		if cmd.ExitCode != 0 {
			statusMsg = fmt.Sprintf("Failed (exit code %d)", cmd.ExitCode)
		}

		result.WriteString(fmt.Sprintf(
			"%d. [%s] $ %s\n   Shell: %s, Duration: %d ms, Status: %s\n\n",
			i+1,
			cmd.StartTime.Format(time.RFC3339),
			cmd.Command,
			cmd.Shell,
			cmd.ExecutionMs,
			statusMsg,
		))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: result.String(),
			},
		},
	}, nil
}

func (s *ShellServer) handleListAllowedCommands(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	if s.allowAllCommands {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "All commands are allowed ('*' mode).\n\nWarning: This server is configured to execute any shell command. This poses a security risk.",
				},
			},
		}, nil
	}

	if len(s.allowedCommands) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "No commands are currently allowed. Configure the server with the '--allowed-commands' flag.",
				},
			},
		}, nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Allowed commands (%d):\n\n", len(s.allowedCommands)))

	for i, cmd := range s.allowedCommands {
		result.WriteString(fmt.Sprintf("%d. %s\n", i+1, cmd))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: result.String(),
			},
		},
	}, nil
}

func (s *ShellServer) Serve() error {
	return server.ServeStdio(s.server)
}

func main() {
	// Parse command line flags
	allowedCommandsFlag := flag.String("allowed-commands", "", "Comma-separated list of allowed commands or '*' to allow all commands")
	flag.Parse()

	if *allowedCommandsFlag == "" {
		fmt.Fprintf(os.Stderr, "Error: The '--allowed-commands' flag is required.\n")
		fmt.Fprintf(os.Stderr, "Usage: %s --allowed-commands=ls,cat,echo,find\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Or to allow all commands (use with caution): %s --allowed-commands=*\n", os.Args[0])
		os.Exit(1)
	}

	// Create and start the server
	shellServer, err := NewShellServer(*allowedCommandsFlag)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Log the server configuration
	if shellServer.allowAllCommands {
		log.Println("Starting shell server with all commands allowed ('*' mode)")
	} else {
		log.Printf("Starting shell server with %d allowed commands", len(shellServer.allowedCommands))
	}

	// Serve requests
	if err := shellServer.Serve(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

# MCP Unix Shell Usage Guide

This guide explains how to use the MCP Unix Shell server with Claude.

## Setup

### With Claude Desktop

1. Install the MCP Unix Shell server:
   ```bash
   go install github.com/gamunu/mcp-unix-shell@latest
   ```

2. Update your Claude Desktop configuration:
   ```json
   {
     "mcpServers": {
       "shell": {
         "command": "mcp-unix-shell",
         "args": [
           "--allowed-commands=ls,cat,echo,find,grep,curl,ps"
         ]
       }
     }
   }
   ```

3. Restart Claude Desktop to apply the changes.

## Examples

Here are some examples of tasks you can accomplish with the MCP Unix Shell server:

### Basic File Operations

```
Please list the files in my home directory

> Using execute_command with "ls ~"
```

### System Information

```
What processes are using the most CPU right now?

> Using execute_command with "ps aux | sort -rk 3,3 | head -n 10"
```

### Network Operations

```
Can you check if my website is up?

> Using execute_command with "curl -I https://example.com"
```

### File Content Analysis

```
What are the most common words in my README file?

> Using execute_command with "cat README.md | tr -s '[:space:]' '\n' | grep -v '^$' | sort | uniq -c | sort -nr | head -n 10"
```

## Security Considerations

When using the MCP Unix Shell, keep these security practices in mind:

1. **Principle of Least Privilege**: Only allow commands that are absolutely necessary.

2. **No Destructive Commands**: Avoid allowing commands that can delete or modify files (like `rm`, `mv`, `chmod`) unless absolutely necessary.

3. **Avoid Privilege Escalation**: Never include commands like `sudo` in your allowed list.

4. **Data Privacy**: Remember that any command output is processed by the AI model, so avoid accessing sensitive data.

5. **Sandbox Usage**: Consider running Claude Desktop in a sandboxed environment with limited permissions when using the shell server.

## Troubleshooting

### Command Not Allowed

If you see an error like:
```
Error: Command 'xyz' is not in the allowed list. Run 'list_allowed_commands' to see what commands are permitted.
```

You need to add the command to your allowed list in the configuration.

### Timeout Errors

Commands have a 30-second execution timeout. For long-running tasks, consider breaking them down into smaller commands or increasing the timeout in the source code.

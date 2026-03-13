package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
)

// Tool describes an MCP tool returned by tools/list.
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required,omitempty"`
}

type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// ContentItem is a single content block in a tool result.
type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ToolResult is the result of a tools/call.
type ToolResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// Client manages an MCP server subprocess over stdio JSON-RPC.
type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	reader *bufio.Reader
	mu     sync.Mutex
	nextID atomic.Int64
}

// NewClient starts the MCP server subprocess with the given root directory.
// serverCmd is the command to run (e.g. "go") and serverArgs are arguments
// (e.g. ["run", "./cmd/mcp-server", "--root", "/path/to/repo"]).
func NewClient(serverCmd string, serverArgs []string) (*Client, error) {
	cmd := exec.Command(serverCmd, serverArgs...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp: stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp: stdout pipe: %w", err)
	}
	cmd.Stderr = nil // discard server stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("mcp: start server: %w", err)
	}
	c := &Client{
		cmd:    cmd,
		stdin:  stdin,
		reader: bufio.NewReaderSize(stdout, 1024*1024),
	}
	return c, nil
}

type jsonRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      *int64      `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (c *Client) call(method string, params interface{}) (json.RawMessage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	id := c.nextID.Add(1)
	req := jsonRPCRequest{JSONRPC: "2.0", ID: &id, Method: method, Params: params}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if _, err := fmt.Fprintf(c.stdin, "%s\n", data); err != nil {
		return nil, fmt.Errorf("mcp: write: %w", err)
	}

	line, err := c.reader.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("mcp: read: %w", err)
	}
	var resp jsonRPCResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, fmt.Errorf("mcp: decode response: %w", err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("mcp: rpc error %d: %s", resp.Error.Code, resp.Error.Message)
	}
	return resp.Result, nil
}

func (c *Client) notify(method string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	req := jsonRPCRequest{JSONRPC: "2.0", Method: method}
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(c.stdin, "%s\n", data)
	return err
}

// Initialize performs the MCP handshake.
func (c *Client) Initialize(_ context.Context) error {
	params := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo":      map[string]string{"name": "agentflow", "version": "1.0.0"},
	}
	_, err := c.call("initialize", params)
	if err != nil {
		return err
	}
	return c.notify("notifications/initialized")
}

// ListTools returns the tools available on the MCP server.
func (c *Client) ListTools(_ context.Context) ([]Tool, error) {
	raw, err := c.call("tools/list", nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Tools []Tool `json:"tools"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("mcp: decode tools: %w", err)
	}
	return result.Tools, nil
}

// CallTool executes a tool on the MCP server and returns the result.
func (c *Client) CallTool(_ context.Context, name string, args map[string]string) (*ToolResult, error) {
	params := map[string]interface{}{
		"name":      name,
		"arguments": args,
	}
	raw, err := c.call("tools/call", params)
	if err != nil {
		return nil, err
	}
	var result ToolResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("mcp: decode tool result: %w", err)
	}
	return &result, nil
}

// Close stops the MCP server subprocess.
func (c *Client) Close() error {
	c.stdin.Close()
	return c.cmd.Wait()
}

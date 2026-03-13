package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const protocolVersion = "2024-11-05"

// --- JSON-RPC types ---

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// --- MCP types ---

type initializeResult struct {
	ProtocolVersion string     `json:"protocolVersion"`
	Capabilities    capability `json:"capabilities"`
	ServerInfo      serverInfo `json:"serverInfo"`
}

type capability struct {
	Tools *struct{} `json:"tools,omitempty"`
}

type serverInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type toolDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema inputSchema `json:"inputSchema"`
}

type inputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]propDef  `json:"properties"`
	Required   []string            `json:"required,omitempty"`
}

type propDef struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type contentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type toolResult struct {
	Content []contentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// --- Server ---

var rootDir string

func main() {
	flag.StringVar(&rootDir, "root", ".", "Root directory for filesystem operations")
	flag.Parse()

	abs, err := filepath.Abs(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid root: %v\n", err)
		os.Exit(1)
	}
	rootDir = abs

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var req jsonRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			continue
		}
		resp := handleRequest(req)
		if resp != nil {
			out, _ := json.Marshal(resp)
			fmt.Fprintf(os.Stdout, "%s\n", out)
		}
	}
}

func handleRequest(req jsonRPCRequest) *jsonRPCResponse {
	switch req.Method {
	case "initialize":
		return &jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: initializeResult{
				ProtocolVersion: protocolVersion,
				Capabilities:    capability{Tools: &struct{}{}},
				ServerInfo:      serverInfo{Name: "agentflow-fs", Version: "1.0.0"},
			},
		}

	case "notifications/initialized":
		return nil

	case "tools/list":
		return &jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"tools": getToolDefinitions(),
			},
		}

	case "tools/call":
		var params toolCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return errorResponse(req.ID, -32602, "invalid params")
		}
		result := executeTool(params)
		return &jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: result}

	default:
		return errorResponse(req.ID, -32601, "method not found: "+req.Method)
	}
}

func errorResponse(id *json.RawMessage, code int, msg string) *jsonRPCResponse {
	return &jsonRPCResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: msg}}
}

func getToolDefinitions() []toolDef {
	return []toolDef{
		{
			Name:        "list_directory",
			Description: "List files and directories at the given path. Returns names with trailing / for directories.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]propDef{
					"path": {Type: "string", Description: "Relative path from repo root (use '.' for root)"},
				},
				Required: []string{"path"},
			},
		},
		{
			Name:        "read_file",
			Description: "Read the contents of a file. Returns the full text content.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]propDef{
					"path": {Type: "string", Description: "Relative file path from repo root"},
				},
				Required: []string{"path"},
			},
		},
		{
			Name:        "write_file",
			Description: "Create or overwrite a file with the given content. Parent directories are created automatically.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]propDef{
					"path":    {Type: "string", Description: "Relative file path from repo root"},
					"content": {Type: "string", Description: "Full file content to write"},
				},
				Required: []string{"path", "content"},
			},
		},
		{
			Name:        "search_files",
			Description: "Search for a regex pattern across files in the repo. Returns matching file paths and line content.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]propDef{
					"pattern": {Type: "string", Description: "Regex pattern to search for"},
					"path":    {Type: "string", Description: "Subdirectory to search in (optional, defaults to repo root)"},
				},
				Required: []string{"pattern"},
			},
		},
	}
}

func executeTool(params toolCallParams) toolResult {
	var args map[string]string
	if err := json.Unmarshal(params.Arguments, &args); err != nil {
		return toolError("invalid arguments: " + err.Error())
	}

	switch params.Name {
	case "list_directory":
		return toolListDir(args["path"])
	case "read_file":
		return toolReadFile(args["path"])
	case "write_file":
		return toolWriteFile(args["path"], args["content"])
	case "search_files":
		return toolSearchFiles(args["pattern"], args["path"])
	default:
		return toolError("unknown tool: " + params.Name)
	}
}

func toolError(msg string) toolResult {
	return toolResult{Content: []contentItem{{Type: "text", Text: msg}}, IsError: true}
}

func toolText(text string) toolResult {
	return toolResult{Content: []contentItem{{Type: "text", Text: text}}}
}

// --- Tool implementations ---

var skipDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, "__pycache__": true,
	".idea": true, ".vscode": true, "dist": true, "build": true,
}

func safePath(rel string) (string, error) {
	if rel == "" {
		rel = "."
	}
	cleaned := filepath.Clean(rel)
	if strings.HasPrefix(cleaned, "..") {
		return "", fmt.Errorf("path traversal not allowed: %s", rel)
	}
	abs := filepath.Join(rootDir, cleaned)
	if !strings.HasPrefix(abs, rootDir) {
		return "", fmt.Errorf("path outside root: %s", rel)
	}
	return abs, nil
}

func toolListDir(path string) toolResult {
	abs, err := safePath(path)
	if err != nil {
		return toolError(err.Error())
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return toolError(err.Error())
	}
	var lines []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			if skipDirs[name] {
				continue
			}
			lines = append(lines, name+"/")
		} else {
			lines = append(lines, name)
		}
	}
	return toolText(strings.Join(lines, "\n"))
}

func toolReadFile(path string) toolResult {
	abs, err := safePath(path)
	if err != nil {
		return toolError(err.Error())
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return toolError(err.Error())
	}
	content := string(data)
	if len(content) > 50000 {
		content = content[:50000] + "\n... (truncated at 50KB)"
	}
	return toolText(content)
}

func toolWriteFile(path, content string) toolResult {
	abs, err := safePath(path)
	if err != nil {
		return toolError(err.Error())
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		return toolError(err.Error())
	}
	if err := os.WriteFile(abs, []byte(content), 0644); err != nil {
		return toolError(err.Error())
	}
	return toolText(fmt.Sprintf("wrote %d bytes to %s", len(content), path))
}

func toolSearchFiles(pattern, subPath string) toolResult {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return toolError("invalid regex: " + err.Error())
	}
	searchRoot, err := safePath(subPath)
	if err != nil {
		return toolError(err.Error())
	}
	var matches []string
	count := 0
	maxMatches := 50

	_ = filepath.Walk(searchRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Size() > 500000 {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(rootDir, path)
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if re.MatchString(line) {
				matches = append(matches, fmt.Sprintf("%s:%d: %s", rel, i+1, strings.TrimSpace(line)))
				count++
				if count >= maxMatches {
					return fmt.Errorf("limit")
				}
			}
		}
		return nil
	})

	if len(matches) == 0 {
		return toolText("no matches found")
	}
	result := strings.Join(matches, "\n")
	if count >= maxMatches {
		result += fmt.Sprintf("\n... (showing first %d matches)", maxMatches)
	}
	return toolText(result)
}

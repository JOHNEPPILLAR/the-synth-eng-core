package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	moduleRoot, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get current working directory: %v", err)
	}

	serverPath := filepath.Join(moduleRoot, "cmd", "pandoras-box")
	c, err := client.NewStdioMCPClient("go", nil, "run", serverPath)
	if err != nil {
		log.Fatalf("failed to create stdio client: %v", err)
	}
	defer c.Close()

	_, err = c.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			Capabilities:    mcp.ClientCapabilities{},
			ClientInfo: mcp.Implementation{
				Name:    "pandoras-box-smoke",
				Version: "0.1.0",
			},
		},
	})
	if err != nil {
		log.Fatalf("initialize failed: %v", err)
	}

	tools, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		log.Fatalf("list tools failed: %v", err)
	}

	requiredTools := []string{
		"environment_info",
		"confluence_search_pages",
		"confluence_get_page",
		"list_github_repos",
		"get_github_file_content",
		"search_github_repo_patterns",
	}

	available := map[string]struct{}{}
	for _, t := range tools.Tools {
		available[t.Name] = struct{}{}
	}
	for _, name := range requiredTools {
		if _, ok := available[name]; !ok {
			log.Fatalf("required tool missing: %s", name)
		}
	}

	fmt.Printf("Discovered %d tools\n", len(tools.Tools))
	printToolNames(available)

	mustCall(ctx, c, "environment_info", map[string]any{})

	if isGitHubConfigured() {
		mustCall(ctx, c, "list_github_repos", map[string]any{"limit": 5})
		mustCall(ctx, c, "get_github_file_content", map[string]any{
			"repo": smokeRepo(),
			"path": "README.md",
		})
		mustCall(ctx, c, "search_github_repo_patterns", map[string]any{
			"repo":        smokeRepo(),
			"pattern":     "module",
			"regex":       false,
			"file_glob":   "**/*.md",
			"max_results": 3,
		})
		fmt.Println("GitHub repo smoke calls executed")
	} else {
		fmt.Println("GitHub token not configured; skipping GitHub repo smoke calls")
	}

	if isConfluenceConfigured() {
		mustCall(ctx, c, "confluence_search_pages", map[string]any{
			"query": "text~\"coding standards\" and type=page",
			"limit": 1,
		})
		fmt.Println("Confluence search smoke test executed")
	} else {
		fmt.Println("Confluence credentials not configured; skipping Confluence smoke calls")
	}

	_ = moduleRoot
	fmt.Println("Pandora's Box MCP smoke test passed")
}

func mustCall(ctx context.Context, c *client.Client, name string, args map[string]any) {
	_, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	})
	if err != nil {
		log.Fatalf("tool call failed (%s): %v", name, err)
	}
	fmt.Printf("Tool call ok: %s\n", name)
}

func printToolNames(available map[string]struct{}) {
	names := make([]string, 0, len(available))
	for name := range available {
		names = append(names, name)
	}
	sort.Strings(names)
	fmt.Printf("Tools: %s\n", strings.Join(names, ", "))
}

func isConfluenceConfigured() bool {
	hasBase := strings.TrimSpace(os.Getenv("CONFLUENCE_BASE_URL")) != ""
	hasBasic := strings.TrimSpace(os.Getenv("CONFLUENCE_EMAIL")) != "" && strings.TrimSpace(os.Getenv("CONFLUENCE_API_TOKEN")) != ""
	hasBearer := strings.TrimSpace(os.Getenv("CONFLUENCE_TOKEN")) != ""
	return hasBase && (hasBasic || hasBearer)
}

func isGitHubConfigured() bool {
	return strings.TrimSpace(os.Getenv("GITHUB_TOKEN")) != "" || strings.TrimSpace(os.Getenv("GH_TOKEN")) != ""
}

func smokeRepo() string {
	if repo := strings.TrimSpace(os.Getenv("GITHUB_SMOKE_REPO")); repo != "" {
		return repo
	}
	return "JOHNEPPILLAR/the-synth-eng-core"
}

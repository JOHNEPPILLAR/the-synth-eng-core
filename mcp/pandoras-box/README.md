# Pandora's Box MCP Server (Go)

MCP server for retrieving engineering context, including:

- Confluence pages for coding standards and platform docs
- GitHub repositories that can be mined for implementation patterns through the GitHub API

## Tools

- `environment_info`
  - Returns local runtime metadata plus Confluence and GitHub config status.

- `confluence_search_pages`
  - Inputs: `query` (required), `space_key` (optional), `limit` (optional)
  - Searches Confluence via CQL.

- `confluence_get_page`
  - Inputs: `page_id` (required)
  - Fetches page metadata and storage body.

- `list_github_repos`
  - Inputs: `visibility` (optional), `affiliation` (optional), `limit` (optional)
  - Lists repositories visible to the configured GitHub token.

- `get_github_file_content`
  - Inputs: `repo` + `path` (+ optional `ref`) or `github_url`
  - Fetches file contents from a GitHub repository, including `SKILL.md`, examples, and reference source files.

- `search_github_repo_patterns`
  - Inputs: `repo` (required, `owner/repo`), `pattern` (required), `regex` (optional), `file_glob` (optional), `max_results` (optional)
  - Searches a GitHub repository by reading repository blobs through the GitHub API.

## Environment Variables

- `CONFLUENCE_BASE_URL` (example: `https://your-org.atlassian.net`)
- Authentication:
  - Option A: `CONFLUENCE_EMAIL` + `CONFLUENCE_API_TOKEN`
  - Option B: `CONFLUENCE_TOKEN` (Bearer)
- `GITHUB_TOKEN` (required for GitHub repo tools; `GH_TOKEN` also works)
- `GITHUB_API_URL` (optional, defaults to `https://api.github.com`)
- `GITHUB_SMOKE_REPO` (optional for smoke test; defaults to `JOHNEPPILLAR/the-synth-eng-core`)

## Run

```bash
go mod tidy
go run ./cmd/pandoras-box
```

## Build

```bash
go build ./cmd/pandoras-box
```

## Smoke Test

Run a lightweight MCP integration check that initializes the server, verifies tool discovery, and calls key tools.

```bash
go run ./cmd/pandoras-box-smoke
```

Notes:

- Confluence smoke calls run only when Confluence environment variables are configured.
- GitHub repo smoke calls run only when a GitHub token is configured.
- `get_github_file_content` is the intended path for remote skill files and code-reference content hosted on github.com.

## Example MCP Client Entry

```json
{
  "mcpServers": {
    "pandoras-box": {
      "command": "go",
      "args": [
        "run",
        "/absolute/path/to/the-synth-eng-core/mcp/pandoras-box/cmd/pandoras-box"
      ],
      "env": {
        "CONFLUENCE_BASE_URL": "https://your-org.atlassian.net",
        "CONFLUENCE_EMAIL": "you@example.com",
        "CONFLUENCE_API_TOKEN": "your_api_token",
        "GITHUB_TOKEN": "your_github_token",
        "GITHUB_API_URL": "https://api.github.com"
      }
    }
  }
}
```

## VS Code Workspace MCP Entry

This repository already includes a workspace config in [.vscode/mcp.json](.vscode/mcp.json) with a `pandoras-box` server entry:

- command: `go run ./mcp/pandoras-box/cmd/pandoras-box`
- cwd: workspace root
- optional env inherited from the VS Code process, including `GITHUB_TOKEN`

package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	defaultConfluenceLimit = 10
	defaultGitHubRepoLimit = 25
	defaultPatternLimit    = 30
	maxFileBytes           = 1024 * 1024
	githubAPIVersion       = "2022-11-28"
	defaultGitHubAPIBase   = "https://api.github.com"
)

type confluenceSearchResponse struct {
	Results []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		Type  string `json:"type"`
		Space struct {
			Key string `json:"key"`
		} `json:"space"`
		Links struct {
			WebUI string `json:"webui"`
		} `json:"_links"`
	} `json:"results"`
	Links struct {
		Base string `json:"base"`
	} `json:"_links"`
}

type confluencePageResponse struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Type  string `json:"type"`
	Space struct {
		Key string `json:"key"`
	} `json:"space"`
	Version struct {
		Number int `json:"number"`
	} `json:"version"`
	Body struct {
		Storage struct {
			Value string `json:"value"`
		} `json:"storage"`
	} `json:"body"`
	Links struct {
		Base  string `json:"base"`
		WebUI string `json:"webui"`
	} `json:"_links"`
}

type githubRepoResponse struct {
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	HTMLURL       string `json:"html_url"`
	Description   string `json:"description"`
	Private       bool   `json:"private"`
	Visibility    string `json:"visibility"`
	DefaultBranch string `json:"default_branch"`
	UpdatedAt     string `json:"updated_at"`
	Owner         struct {
		Login string `json:"login"`
	} `json:"owner"`
}

type githubTreeResponse struct {
	Truncated bool `json:"truncated"`
	Tree      []struct {
		Path string `json:"path"`
		Type string `json:"type"`
		SHA  string `json:"sha"`
		Size int    `json:"size"`
	} `json:"tree"`
}

type githubBlobResponse struct {
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
	Size     int    `json:"size"`
}

type githubContentResponse struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	SHA         string `json:"sha"`
	Size        int    `json:"size"`
	HTMLURL     string `json:"html_url"`
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"`
	Content     string `json:"content"`
	Encoding    string `json:"encoding"`
}

type githubUserResponse struct {
	Login   string `json:"login"`
	HTMLURL string `json:"html_url"`
}

type repoMatch struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Snippet string `json:"snippet"`
}

type confluenceClient struct {
	baseURL    string
	email      string
	apiToken   string
	bearerAuth string
	httpClient *http.Client
}

type githubClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

type lineMatcher interface {
	MatchString(s string) bool
}

type containsMatcher struct {
	needle string
}

func (m containsMatcher) MatchString(s string) bool {
	return strings.Contains(s, m.needle)
}

func main() {
	s := server.NewMCPServer(
		"Pandora's Box",
		"0.2.0",
		server.WithToolCapabilities(false),
	)

	s.AddTool(newEnvironmentInfoTool(), environmentInfoHandler)
	s.AddTool(newConfluenceSearchTool(), confluenceSearchHandler)
	s.AddTool(newConfluenceGetPageTool(), confluenceGetPageHandler)
	s.AddTool(newListGitHubReposTool(), listGitHubReposHandler)
	s.AddTool(newGetGitHubFileContentTool(), getGitHubFileContentHandler)
	s.AddTool(newSearchGitHubRepoPatternsTool(), searchGitHubRepoPatternsHandler)

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "mcp server exited with error: %v\n", err)
		os.Exit(1)
	}
}

func newEnvironmentInfoTool() mcp.Tool {
	return mcp.NewTool("environment_info",
		mcp.WithDescription("Return local runtime metadata plus Confluence and GitHub access status."),
	)
}

func environmentInfoHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	_ = req

	currentUser := ""
	if u, err := user.Current(); err == nil {
		currentUser = u.Username
	}

	cwd, _ := os.Getwd()
	result := map[string]any{
		"local_user":            currentUser,
		"os":                    runtime.GOOS,
		"arch":                  runtime.GOARCH,
		"working_directory":     cwd,
		"confluence_configured": isConfluenceConfigured(),
		"github_configured":     isGitHubConfigured(),
		"github_api_base":       currentGitHubAPIBase(),
	}

	if client, err := newGitHubClientFromEnv(); err == nil {
		viewer, viewerErr := client.currentUser(ctx)
		if viewerErr == nil {
			result["github_user"] = viewer.Login
			result["github_user_url"] = viewer.HTMLURL
		} else {
			result["github_status"] = viewerErr.Error()
		}
	}

	return structuredResult(result, "environment info retrieved")
}

func newConfluenceSearchTool() mcp.Tool {
	return mcp.NewTool("confluence_search_pages",
		mcp.WithDescription("Search Confluence pages via CQL. Requires CONFLUENCE_BASE_URL and auth env vars."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("CQL query. Example: text~\"coding standards\" and type=page"),
		),
		mcp.WithString("space_key",
			mcp.Description("Optional Confluence space key to constrain search."),
		),
		mcp.WithNumber("limit",
			mcp.Description("Max results to return. Default 10."),
		),
	)
}

func confluenceSearchHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	spaceKey, _ := req.RequireString("space_key")
	limit := numberArg(req, "limit", defaultConfluenceLimit)
	if limit <= 0 {
		limit = defaultConfluenceLimit
	}

	client, err := newConfluenceClientFromEnv()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	pages, err := client.searchPages(ctx, query, spaceKey, limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return structuredResult(map[string]any{"results": pages}, fmt.Sprintf("found %d pages", len(pages)))
}

func newConfluenceGetPageTool() mcp.Tool {
	return mcp.NewTool("confluence_get_page",
		mcp.WithDescription("Get Confluence page details by page ID, including content storage body."),
		mcp.WithString("page_id",
			mcp.Required(),
			mcp.Description("Confluence content/page ID."),
		),
	)
}

func confluenceGetPageHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pageID, err := req.RequireString("page_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	client, err := newConfluenceClientFromEnv()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	page, err := client.getPage(ctx, pageID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return structuredResult(page, "page retrieved")
}

func newListGitHubReposTool() mcp.Tool {
	return mcp.NewTool("list_github_repos",
		mcp.WithDescription("List repositories visible to the authenticated GitHub token on github.com."),
		mcp.WithString("visibility",
			mcp.Description("Optional visibility filter: all, public, or private. Default all."),
		),
		mcp.WithString("affiliation",
			mcp.Description("Optional affiliation filter. Default owner,collaborator,organization_member."),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum repositories to return. Default 25."),
		),
	)
}

func newGetGitHubFileContentTool() mcp.Tool {
	return mcp.NewTool("get_github_file_content",
		mcp.WithDescription("Fetch file contents from a GitHub repository using GITHUB_TOKEN. Accepts either repo/path/ref or a github_url to a blob page."),
		mcp.WithString("repo",
			mcp.Description("GitHub repository in owner/repo format. Optional when github_url is provided."),
		),
		mcp.WithString("path",
			mcp.Description("Repository file path. Optional when github_url is provided."),
		),
		mcp.WithString("ref",
			mcp.Description("Optional branch, tag, or commit SHA. Defaults to the repository default branch."),
		),
		mcp.WithString("github_url",
			mcp.Description("Optional https://github.com/<owner>/<repo>/blob/<ref>/<path> URL."),
		),
	)
}

func listGitHubReposHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	visibility, _ := req.RequireString("visibility")
	affiliation, _ := req.RequireString("affiliation")
	limit := numberArg(req, "limit", defaultGitHubRepoLimit)
	if limit <= 0 {
		limit = defaultGitHubRepoLimit
	}

	client, err := newGitHubClientFromEnv()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	repos, err := client.listRepositories(ctx, visibility, affiliation, limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return structuredResult(map[string]any{
		"visibility":  defaultString(visibility, "all"),
		"affiliation": defaultString(affiliation, "owner,collaborator,organization_member"),
		"repos":       repos,
	}, fmt.Sprintf("found %d repositories", len(repos)))
}

func getGitHubFileContentHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repoName, _ := req.RequireString("repo")
	path, _ := req.RequireString("path")
	ref, _ := req.RequireString("ref")
	githubURL, _ := req.RequireString("github_url")

	if strings.TrimSpace(githubURL) != "" {
		parsedRepo, parsedPath, parsedRef, err := parseGitHubBlobURL(githubURL)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if strings.TrimSpace(repoName) == "" {
			repoName = parsedRepo
		}
		if strings.TrimSpace(path) == "" {
			path = parsedPath
		}
		if strings.TrimSpace(ref) == "" {
			ref = parsedRef
		}
	}

	if strings.TrimSpace(repoName) == "" || strings.TrimSpace(path) == "" {
		return mcp.NewToolResultError("repo and path are required unless github_url provides both"), nil
	}

	client, err := newGitHubClientFromEnv()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	content, err := client.getFileContent(ctx, repoName, path, ref)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return structuredResult(content, "github file content retrieved")
}

func newSearchGitHubRepoPatternsTool() mcp.Tool {
	return mcp.NewTool("search_github_repo_patterns",
		mcp.WithDescription("Search files in a GitHub repository by downloading repository blobs through the GitHub API. Requires GITHUB_TOKEN."),
		mcp.WithString("repo",
			mcp.Required(),
			mcp.Description("GitHub repository in owner/repo format or a github.com repository URL."),
		),
		mcp.WithString("pattern",
			mcp.Required(),
			mcp.Description("Regex or plain-text pattern to search for in repository files."),
		),
		mcp.WithBoolean("regex",
			mcp.Description("Set false to do plain-text search. Default true."),
		),
		mcp.WithString("file_glob",
			mcp.Description("Optional glob filter, e.g. **/*.md or **/*.go."),
		),
		mcp.WithNumber("max_results",
			mcp.Description("Maximum matches to return. Default 30."),
		),
	)
}

func searchGitHubRepoPatternsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repoName, err := req.RequireString("repo")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	pattern, err := req.RequireString("pattern")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	glob, _ := req.RequireString("file_glob")
	maxResults := numberArg(req, "max_results", defaultPatternLimit)
	if maxResults <= 0 {
		maxResults = defaultPatternLimit
	}

	isRegex := true
	if v, ok := req.GetArguments()["regex"]; ok {
		if b, ok := v.(bool); ok {
			isRegex = b
		}
	}

	client, err := newGitHubClientFromEnv()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := client.searchRepoPatterns(ctx, repoName, pattern, glob, isRegex, maxResults)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return structuredResult(result, fmt.Sprintf("found %d matches", len(result["matches"].([]repoMatch))))
}

func newConfluenceClientFromEnv() (*confluenceClient, error) {
	base := strings.TrimSpace(os.Getenv("CONFLUENCE_BASE_URL"))
	if base == "" {
		return nil, errors.New("missing CONFLUENCE_BASE_URL")
	}

	email := strings.TrimSpace(os.Getenv("CONFLUENCE_EMAIL"))
	apiToken := strings.TrimSpace(os.Getenv("CONFLUENCE_API_TOKEN"))
	bearer := strings.TrimSpace(os.Getenv("CONFLUENCE_TOKEN"))
	if (email == "" || apiToken == "") && bearer == "" {
		return nil, errors.New("configure CONFLUENCE_EMAIL + CONFLUENCE_API_TOKEN, or CONFLUENCE_TOKEN")
	}

	return &confluenceClient{
		baseURL:    strings.TrimRight(base, "/"),
		email:      email,
		apiToken:   apiToken,
		bearerAuth: bearer,
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}, nil
}

func newGitHubClientFromEnv() (*githubClient, error) {
	token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	if token == "" {
		token = strings.TrimSpace(os.Getenv("GH_TOKEN"))
	}
	if token == "" {
		return nil, errors.New("missing GITHUB_TOKEN (or GH_TOKEN) for GitHub API access")
	}

	return &githubClient{
		baseURL:    currentGitHubAPIBase(),
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (c *confluenceClient) searchPages(ctx context.Context, query, spaceKey string, limit int) ([]map[string]any, error) {
	cql := strings.TrimSpace(query)
	if spaceKey != "" {
		cql = fmt.Sprintf("(%s) and space=%s", cql, strconv.Quote(spaceKey))
	}

	u, err := url.Parse(c.baseURL + "/wiki/rest/api/content/search")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("cql", cql)
	q.Set("limit", strconv.Itoa(limit))
	u.RawQuery = q.Encode()

	var response confluenceSearchResponse
	if err := c.getJSON(ctx, u.String(), &response); err != nil {
		return nil, err
	}

	out := make([]map[string]any, 0, len(response.Results))
	for _, page := range response.Results {
		webURL := strings.TrimRight(response.Links.Base, "/") + page.Links.WebUI
		out = append(out, map[string]any{
			"id":        page.ID,
			"title":     page.Title,
			"type":      page.Type,
			"space_key": page.Space.Key,
			"url":       webURL,
		})
	}
	return out, nil
}

func (c *confluenceClient) getPage(ctx context.Context, pageID string) (map[string]any, error) {
	u, err := url.Parse(c.baseURL + "/wiki/rest/api/content/" + url.PathEscape(pageID))
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("expand", "space,version,body.storage")
	u.RawQuery = q.Encode()

	var response confluencePageResponse
	if err := c.getJSON(ctx, u.String(), &response); err != nil {
		return nil, err
	}

	return map[string]any{
		"id":             response.ID,
		"title":          response.Title,
		"type":           response.Type,
		"space_key":      response.Space.Key,
		"version":        response.Version.Number,
		"url":            strings.TrimRight(response.Links.Base, "/") + response.Links.WebUI,
		"body_storage":   response.Body.Storage.Value,
		"body_truncated": len(response.Body.Storage.Value) > 4000,
	}, nil
}

func (c *confluenceClient) getJSON(ctx context.Context, endpoint string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if c.bearerAuth != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearerAuth)
	} else {
		token := base64.StdEncoding.EncodeToString([]byte(c.email + ":" + c.apiToken))
		req.Header.Set("Authorization", "Basic "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("confluence request failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return err
	}
	return nil
}

func (c *githubClient) currentUser(ctx context.Context) (githubUserResponse, error) {
	var response githubUserResponse
	if err := c.getJSON(ctx, c.baseURL+"/user", &response); err != nil {
		return githubUserResponse{}, err
	}
	return response, nil
}

func (c *githubClient) listRepositories(ctx context.Context, visibility, affiliation string, limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = defaultGitHubRepoLimit
	}

	repos := make([]map[string]any, 0, limit)
	page := 1
	for len(repos) < limit {
		perPage := limit - len(repos)
		if perPage > 100 {
			perPage = 100
		}

		u, err := url.Parse(c.baseURL + "/user/repos")
		if err != nil {
			return nil, err
		}
		q := u.Query()
		q.Set("sort", "updated")
		q.Set("direction", "desc")
		q.Set("per_page", strconv.Itoa(perPage))
		q.Set("page", strconv.Itoa(page))
		if strings.TrimSpace(visibility) != "" {
			q.Set("visibility", visibility)
		}
		if strings.TrimSpace(affiliation) != "" {
			q.Set("affiliation", affiliation)
		}
		u.RawQuery = q.Encode()

		var response []githubRepoResponse
		if err := c.getJSON(ctx, u.String(), &response); err != nil {
			return nil, err
		}
		if len(response) == 0 {
			break
		}

		for _, repo := range response {
			repos = append(repos, map[string]any{
				"name":           repo.Name,
				"full_name":      repo.FullName,
				"owner":          repo.Owner.Login,
				"url":            repo.HTMLURL,
				"description":    repo.Description,
				"private":        repo.Private,
				"visibility":     repo.Visibility,
				"default_branch": repo.DefaultBranch,
				"updated_at":     repo.UpdatedAt,
			})
			if len(repos) >= limit {
				break
			}
		}

		if len(response) < perPage {
			break
		}
		page++
	}

	return repos, nil
}

func (c *githubClient) searchRepoPatterns(ctx context.Context, repoName, pattern, fileGlob string, isRegex bool, maxResults int) (map[string]any, error) {
	owner, repo, err := parseRepoName(repoName)
	if err != nil {
		return nil, err
	}

	metadata, err := c.getRepository(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	tree, err := c.getRepositoryTree(ctx, owner, repo, metadata.DefaultBranch)
	if err != nil {
		return nil, err
	}

	matcher, err := buildMatcher(pattern, isRegex)
	if err != nil {
		return nil, err
	}

	globPattern := strings.TrimSpace(fileGlob)
	if globPattern == "" {
		globPattern = "**/*"
	}

	matches := make([]repoMatch, 0, maxResults)
	filesScanned := 0
	for _, entry := range tree.Tree {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if entry.Type != "blob" {
			continue
		}
		if entry.Size > maxFileBytes {
			continue
		}
		if !matchesGlob(entry.Path, globPattern) {
			continue
		}

		content, err := c.getBlobContent(ctx, owner, repo, entry.SHA)
		if err != nil {
			continue
		}
		filesScanned++

		fileMatches := searchContent(entry.Path, content, matcher, maxResults-len(matches))
		matches = append(matches, fileMatches...)
		if len(matches) >= maxResults {
			break
		}
	}

	return map[string]any{
		"repo":           metadata.FullName,
		"repo_url":       metadata.HTMLURL,
		"default_branch": metadata.DefaultBranch,
		"matches":        matches,
		"files_scanned":  filesScanned,
		"tree_truncated": tree.Truncated,
		"pattern":        pattern,
		"regex":          isRegex,
		"file_glob":      globPattern,
	}, nil
}

func (c *githubClient) getFileContent(ctx context.Context, repoName, filePath, ref string) (map[string]any, error) {
	owner, repo, err := parseRepoName(repoName)
	if err != nil {
		return nil, err
	}

	cleanPath := strings.Trim(strings.TrimSpace(filePath), "/")
	if cleanPath == "" {
		return nil, errors.New("path is required")
	}

	if strings.TrimSpace(ref) == "" {
		metadata, metadataErr := c.getRepository(ctx, owner, repo)
		if metadataErr != nil {
			return nil, metadataErr
		}
		ref = metadata.DefaultBranch
	}

	u, err := url.Parse(fmt.Sprintf("%s/repos/%s/%s/contents/%s", c.baseURL, url.PathEscape(owner), url.PathEscape(repo), cleanPath))
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("ref", ref)
	u.RawQuery = q.Encode()

	var response githubContentResponse
	if err := c.getJSON(ctx, u.String(), &response); err != nil {
		return nil, err
	}
	if response.Type != "file" {
		return nil, fmt.Errorf("%s is not a file", cleanPath)
	}
	if response.Encoding != "base64" {
		return nil, fmt.Errorf("unsupported content encoding: %s", response.Encoding)
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(response.Content, "\n", ""))
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"repo":         owner + "/" + repo,
		"path":         response.Path,
		"ref":          ref,
		"name":         response.Name,
		"size":         response.Size,
		"sha":          response.SHA,
		"url":          response.HTMLURL,
		"download_url": response.DownloadURL,
		"content":      string(decoded),
	}, nil
}

func (c *githubClient) getRepository(ctx context.Context, owner, repo string) (githubRepoResponse, error) {
	var response githubRepoResponse
	endpoint := fmt.Sprintf("%s/repos/%s/%s", c.baseURL, url.PathEscape(owner), url.PathEscape(repo))
	if err := c.getJSON(ctx, endpoint, &response); err != nil {
		return githubRepoResponse{}, err
	}
	return response, nil
}

func (c *githubClient) getRepositoryTree(ctx context.Context, owner, repo, ref string) (githubTreeResponse, error) {
	u, err := url.Parse(fmt.Sprintf("%s/repos/%s/%s/git/trees/%s", c.baseURL, url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(ref)))
	if err != nil {
		return githubTreeResponse{}, err
	}
	q := u.Query()
	q.Set("recursive", "1")
	u.RawQuery = q.Encode()

	var response githubTreeResponse
	if err := c.getJSON(ctx, u.String(), &response); err != nil {
		return githubTreeResponse{}, err
	}
	return response, nil
}

func (c *githubClient) getBlobContent(ctx context.Context, owner, repo, sha string) (string, error) {
	var response githubBlobResponse
	endpoint := fmt.Sprintf("%s/repos/%s/%s/git/blobs/%s", c.baseURL, url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(sha))
	if err := c.getJSON(ctx, endpoint, &response); err != nil {
		return "", err
	}
	if response.Encoding != "base64" {
		return "", fmt.Errorf("unsupported blob encoding: %s", response.Encoding)
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(response.Content, "\n", ""))
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func (c *githubClient) getJSON(ctx context.Context, endpoint string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("X-GitHub-Api-Version", githubAPIVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("github request failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return err
	}
	return nil
}

func buildMatcher(pattern string, isRegex bool) (lineMatcher, error) {
	if !isRegex {
		return containsMatcher{needle: pattern}, nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}
	return re, nil
}

func searchContent(path, content string, matcher lineMatcher, limit int) []repoMatch {
	if limit <= 0 {
		return nil
	}

	lines := strings.Split(content, "\n")
	results := make([]repoMatch, 0, limit)
	for index, line := range lines {
		line = strings.TrimRight(line, "\r")
		if matcher.MatchString(line) {
			results = append(results, repoMatch{
				Path:    path,
				Line:    index + 1,
				Snippet: strings.TrimSpace(line),
			})
			if len(results) >= limit {
				break
			}
		}
	}
	return results
}

func matchesGlob(path, globPattern string) bool {
	path = filepath.ToSlash(strings.TrimSpace(path))
	globPattern = filepath.ToSlash(strings.TrimSpace(globPattern))

	if globPattern == "" || globPattern == "**/*" {
		return true
	}

	if strings.HasPrefix(globPattern, "**/") {
		tail := strings.TrimPrefix(globPattern, "**/")
		if matched, _ := filepath.Match(tail, filepath.Base(path)); matched {
			return true
		}
	}

	if matched, _ := filepath.Match(globPattern, path); matched {
		return true
	}

	if strings.HasPrefix(globPattern, "*.") {
		if matched, _ := filepath.Match(globPattern, filepath.Base(path)); matched {
			return true
		}
	}

	return false
}

func parseRepoName(input string) (string, string, error) {
	trimmed := strings.TrimSpace(input)
	trimmed = strings.TrimPrefix(trimmed, "https://github.com/")
	trimmed = strings.TrimPrefix(trimmed, "http://github.com/")
	trimmed = strings.TrimPrefix(trimmed, "github.com/")
	trimmed = strings.TrimSuffix(trimmed, ".git")
	trimmed = strings.Trim(trimmed, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("repo must be in owner/repo format")
	}
	return parts[0], parts[1], nil
}

func parseGitHubBlobURL(input string) (string, string, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(input))
	if err != nil {
		return "", "", "", err
	}
	if !strings.EqualFold(parsed.Host, "github.com") {
		return "", "", "", fmt.Errorf("github_url must point to github.com")
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 5 || parts[2] != "blob" {
		return "", "", "", fmt.Errorf("github_url must look like https://github.com/<owner>/<repo>/blob/<ref>/<path>")
	}

	repo := parts[0] + "/" + parts[1]
	ref := parts[3]
	path := strings.Join(parts[4:], "/")
	if path == "" {
		return "", "", "", fmt.Errorf("github_url must include a file path")
	}

	return repo, path, ref, nil
}

func isConfluenceConfigured() bool {
	base := strings.TrimSpace(os.Getenv("CONFLUENCE_BASE_URL")) != ""
	basic := strings.TrimSpace(os.Getenv("CONFLUENCE_EMAIL")) != "" && strings.TrimSpace(os.Getenv("CONFLUENCE_API_TOKEN")) != ""
	bearer := strings.TrimSpace(os.Getenv("CONFLUENCE_TOKEN")) != ""
	return base && (basic || bearer)
}

func isGitHubConfigured() bool {
	return strings.TrimSpace(os.Getenv("GITHUB_TOKEN")) != "" || strings.TrimSpace(os.Getenv("GH_TOKEN")) != ""
}

func currentGitHubAPIBase() string {
	base := strings.TrimSpace(os.Getenv("GITHUB_API_URL"))
	if base == "" {
		return defaultGitHubAPIBase
	}
	return strings.TrimRight(base, "/")
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func numberArg(req mcp.CallToolRequest, name string, fallback int) int {
	value, ok := req.GetArguments()[name]
	if !ok {
		return fallback
	}
	switch n := value.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case json.Number:
		i, err := n.Int64()
		if err == nil {
			return int(i)
		}
	}
	return fallback
}

func structuredResult(v any, fallback string) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultStructured(v, fallback), nil
}

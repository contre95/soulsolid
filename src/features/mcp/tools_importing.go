package mcp

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *Service) registerImportingTools(srv *server.MCPServer) {
	s.addWriteTool(srv, mcpgo.NewTool("import_path",
		mcpgo.WithDescription("Trigger an import job for a directory inside the configured download path. Returns a job ID; use get_job to track progress."),
		mcpgo.WithString("path", mcpgo.Description("Directory to import: relative to the download path, or an absolute path inside it. Omit to import the whole download path.")),
	), s.importPath)

	s.addReadTool(srv, mcpgo.NewTool("list_import_queue",
		mcpgo.WithDescription("List all items currently in the import queue (tracks awaiting review before being added to the library)"),
	), s.listImportQueue)

	s.addDestructiveTool(srv, mcpgo.NewTool("resolve_queue_item",
		mcpgo.WithDescription("Resolve an import queue item. Actions: 'import' (add to library), 'replace' (replace existing duplicate), 'delete' (remove the file), 'skip' (dismiss)."),
		mcpgo.WithString("id", mcpgo.Required(), mcpgo.Description("Queue item ID (from list_import_queue)")),
		mcpgo.WithString("action", mcpgo.Required(), mcpgo.Description("One of: import, replace, delete, skip")),
	), s.resolveQueueItem)

	s.addDestructiveTool(srv, mcpgo.NewTool("resolve_import_group",
		mcpgo.WithDescription("Apply an action to a whole group of import queue items at once (e.g. all tracks from the same album)."),
		mcpgo.WithString("group_type", mcpgo.Required(), mcpgo.Description("Group type: 'album' or 'artist'")),
		mcpgo.WithString("group_key", mcpgo.Required(), mcpgo.Description("Group key value (album title or artist name)")),
		mcpgo.WithString("action", mcpgo.Required(), mcpgo.Description("One of: import, replace, delete, skip")),
	), s.resolveImportGroup)

	s.addDestructiveTool(srv, mcpgo.NewTool("clear_import_queue",
		mcpgo.WithDescription("Remove all items from the import queue without importing them"),
	), s.clearImportQueue)

	s.addReadTool(srv, mcpgo.NewTool("get_watcher_status",
		mcpgo.WithDescription("Get the current status of the filesystem watcher that auto-imports new music files"),
	), s.getWatcherStatus)

	s.addWriteTool(srv, mcpgo.NewTool("toggle_watcher",
		mcpgo.WithDescription("Start or stop the filesystem watcher that auto-imports new music files"),
		mcpgo.WithString("action", mcpgo.Required(), mcpgo.Description("One of: start, stop")),
	), s.toggleWatcher)

	s.addDestructiveTool(srv, mcpgo.NewTool("run_reorganize",
		mcpgo.WithDescription("Start a background job that reorganizes music files according to the configured naming template. Returns a job ID."),
		mcpgo.WithBoolean("fat32_safe", mcpgo.Description("Sanitize paths for FAT32 filesystems (defaults to the configured import.paths.fat32_safe)")),
	), s.runReorganize)
}

func (s *Service) importPath(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	path, err := resolveImportPath(s.config.Get().DownloadPath, strArg(req.Params.Arguments, "path"))
	if err != nil {
		return errResult(err.Error()), nil
	}
	jobID, err := s.importing.ImportDirectory(ctx, path)
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(map[string]string{"job_id": jobID}), nil
}

// resolveImportPath confines path to downloadPath. Imports may move files
// (import.move), so an agent must never be able to pull files in from elsewhere
// on the server. Symlinks are resolved so one inside the download path cannot
// point outside it.
func resolveImportPath(downloadPath, path string) (string, error) {
	root, err := filepath.EvalSymlinks(downloadPath)
	if err != nil {
		return "", fmt.Errorf("cannot resolve download path: %w", err)
	}
	if path == "" {
		return root, nil
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(downloadPath, path)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("cannot resolve path: %w", err)
	}
	rel, err := filepath.Rel(root, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path must be inside the download path (%s)", downloadPath)
	}
	return resolved, nil
}

func (s *Service) listImportQueue(_ context.Context, _ mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	items := s.importing.GetQueuedItems()
	results := make([]map[string]any, 0, len(items))
	for _, item := range items {
		results = append(results, queueItemSummary(item))
	}
	return jsonResult(results), nil
}

func (s *Service) resolveQueueItem(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := req.Params.Arguments
	if err := s.importing.ProcessQueueItem(ctx, strArg(args, "id"), strArg(args, "action")); err != nil {
		return errResult(err.Error()), nil
	}
	return textResult("queue item resolved"), nil
}

func (s *Service) resolveImportGroup(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := req.Params.Arguments
	if err := s.importing.ProcessQueueGroup(ctx, strArg(args, "group_key"), strArg(args, "group_type"), strArg(args, "action")); err != nil {
		return errResult(err.Error()), nil
	}
	return textResult("import queue group resolved"), nil
}

func (s *Service) clearImportQueue(_ context.Context, _ mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := s.importing.ClearQueue(); err != nil {
		return errResult(err.Error()), nil
	}
	return textResult("import queue cleared"), nil
}

func (s *Service) getWatcherStatus(_ context.Context, _ mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	return jsonResult(map[string]bool{"running": s.importing.GetWatcherStatus()}), nil
}

func (s *Service) toggleWatcher(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	switch strArg(req.Params.Arguments, "action") {
	case "start":
		if err := s.importing.StartWatcher(); err != nil {
			return errResult(err.Error()), nil
		}
		return textResult("file watcher started"), nil
	case "stop":
		if err := s.importing.StopWatcher(); err != nil {
			return errResult(err.Error()), nil
		}
		return textResult("file watcher stopped"), nil
	default:
		return errResult("action must be 'start' or 'stop'"), nil
	}
}

func (s *Service) runReorganize(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	fat32Safe := boolArg(req.Params.Arguments, "fat32_safe", s.config.Get().Import.PathOptions.Fat32Safe)
	jobID, err := s.reorganize.StartReorganizeAnalysis(ctx, fat32Safe)
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(map[string]string{"job_id": jobID}), nil
}

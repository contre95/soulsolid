package mcp

import (
	"context"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *Service) registerImportingTools(srv *server.MCPServer) {
	srv.AddTool(mcpgo.NewTool("import_path",
		mcpgo.WithDescription("Trigger an import job for a directory on the server filesystem. Returns a job ID; use get_job to track progress."),
		mcpgo.WithString("path", mcpgo.Required(), mcpgo.Description("Absolute directory path on the server to import")),
	), s.importPath)

	srv.AddTool(mcpgo.NewTool("list_import_queue",
		mcpgo.WithDescription("List all items currently in the import queue (tracks awaiting review before being added to the library)"),
	), s.listImportQueue)

	srv.AddTool(mcpgo.NewTool("resolve_queue_item",
		mcpgo.WithDescription("Resolve an import queue item. Actions: 'import' (add to library), 'replace' (replace existing duplicate), 'delete' (remove the file), 'skip' (dismiss)."),
		mcpgo.WithString("id", mcpgo.Required(), mcpgo.Description("Queue item ID (from list_import_queue)")),
		mcpgo.WithString("action", mcpgo.Required(), mcpgo.Description("One of: import, replace, delete, skip")),
	), s.resolveQueueItem)

	srv.AddTool(mcpgo.NewTool("resolve_import_group",
		mcpgo.WithDescription("Apply an action to a whole group of import queue items at once (e.g. all tracks from the same album)."),
		mcpgo.WithString("group_type", mcpgo.Required(), mcpgo.Description("Group type: 'album' or 'artist'")),
		mcpgo.WithString("group_key", mcpgo.Required(), mcpgo.Description("Group key value (album title or artist name)")),
		mcpgo.WithString("action", mcpgo.Required(), mcpgo.Description("One of: import, replace, delete, skip")),
	), s.resolveImportGroup)

	srv.AddTool(mcpgo.NewTool("clear_import_queue",
		mcpgo.WithDescription("Remove all items from the import queue without importing them"),
	), s.clearImportQueue)

	srv.AddTool(mcpgo.NewTool("get_watcher_status",
		mcpgo.WithDescription("Get the current status of the filesystem watcher that auto-imports new music files"),
	), s.getWatcherStatus)

	srv.AddTool(mcpgo.NewTool("toggle_watcher",
		mcpgo.WithDescription("Start or stop the filesystem watcher that auto-imports new music files"),
		mcpgo.WithString("action", mcpgo.Required(), mcpgo.Description("One of: start, stop")),
	), s.toggleWatcher)

	srv.AddTool(mcpgo.NewTool("run_reorganize",
		mcpgo.WithDescription("Start a background job that reorganizes music files according to the configured naming template. Returns a job ID."),
		mcpgo.WithBoolean("fat32_safe", mcpgo.Description("Sanitize paths for FAT32 filesystems (defaults to the configured import.paths.fat32_safe)")),
	), s.runReorganize)
}

func (s *Service) importPath(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	jobID, err := s.importing.ImportDirectory(ctx, strArg(req.Params.Arguments, "path"))
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(map[string]string{"job_id": jobID}), nil
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

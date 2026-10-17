package mcp

import (
	"context"
	"os"

	"github.com/contre95/soulsolid/src/music"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *Service) registerJobTools(srv *server.MCPServer) {
	s.addReadTool(srv, mcpgo.NewTool("list_jobs",
		mcpgo.WithDescription("List background jobs (downloads, imports, analysis). Optionally filter by status."),
		mcpgo.WithString("status", mcpgo.Description("Filter by status: pending, running, completed, failed, cancelled")),
	), s.listJobs)

	s.addReadTool(srv, mcpgo.NewTool("get_job",
		mcpgo.WithDescription("Get current status and progress of a background job by ID"),
		mcpgo.WithString("id", mcpgo.Required(), mcpgo.Description("Job ID")),
	), s.getJob)

	s.addReadTool(srv, mcpgo.NewTool("get_job_logs",
		mcpgo.WithDescription("Get the plain-text log output for a background job (the last lines only)"),
		mcpgo.WithString("id", mcpgo.Required(), mcpgo.Description("Job ID")),
		mcpgo.WithNumber("tail", mcpgo.Description("Number of trailing log lines to return (default 200, max 2000)")),
	), s.getJobLogs)

	s.addWriteTool(srv, mcpgo.NewTool("start_job",
		mcpgo.WithDescription("Start a parameterless background job. Only 'calculate_metrics' is supported; downloads, imports and analyses have dedicated tools."),
		mcpgo.WithString("type", mcpgo.Required(), mcpgo.Description("Job type: calculate_metrics")),
	), s.startJob)

	s.addWriteTool(srv, mcpgo.NewTool("cancel_job",
		mcpgo.WithDescription("Cancel a running or pending background job"),
		mcpgo.WithString("id", mcpgo.Required(), mcpgo.Description("Job ID")),
	), s.cancelJob)

	s.addDestructiveTool(srv, mcpgo.NewTool("clear_finished_jobs",
		mcpgo.WithDescription("Remove all completed, failed, and cancelled jobs from the job history"),
	), s.clearFinishedJobs)
}

func (s *Service) listJobs(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	statusFilter := strArg(req.Params.Arguments, "status")
	jobs := s.jobs.GetJobs()
	results := make([]map[string]any, 0, len(jobs))
	for _, job := range jobs {
		if statusFilter != "" && job.Status != music.JobStatus(statusFilter) {
			continue
		}
		results = append(results, jobSummary(job))
	}
	return jsonResult(results), nil
}

func (s *Service) getJob(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	job, exists := s.jobs.GetJob(strArg(req.Params.Arguments, "id"))
	if !exists {
		return errResult("job not found"), nil
	}
	return jsonResult(jobSummary(job)), nil
}

func (s *Service) getJobLogs(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	job, exists := s.jobs.GetJob(strArg(req.Params.Arguments, "id"))
	if !exists {
		return errResult("job not found"), nil
	}
	if job.LogPath == "" {
		return textResult("No logs for this job."), nil
	}
	logContent, err := os.ReadFile(job.LogPath)
	if err != nil {
		return errResult("failed to read log file: " + err.Error()), nil
	}
	tail := min(max(intArg(req.Params.Arguments, "tail", 200), 1), 2000)
	return textResult(truncateText(tailLines(string(logContent), tail))), nil
}

func (s *Service) startJob(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	jobType := strArg(req.Params.Arguments, "type")
	// Other job types take parameters and have dedicated tools that validate
	// them; starting them here would bypass that (and read-only annotations).
	if jobType != "calculate_metrics" {
		return errResult("unsupported job type: only 'calculate_metrics' can be started with start_job"), nil
	}
	jobID, err := s.jobs.StartJob(jobType, jobType, nil)
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(map[string]string{"job_id": jobID}), nil
}

func (s *Service) cancelJob(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := s.jobs.CancelJob(strArg(req.Params.Arguments, "id")); err != nil {
		return errResult(err.Error()), nil
	}
	return textResult("job cancelled"), nil
}

func (s *Service) clearFinishedJobs(_ context.Context, _ mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := s.jobs.ClearFinishedJobs(); err != nil {
		return errResult(err.Error()), nil
	}
	return textResult("finished jobs cleared"), nil
}

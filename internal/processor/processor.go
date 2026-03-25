package processor

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"

	"cli_tool/internal/parser"
	"cli_tool/internal/reporter"
)

type FileReadResult struct {
	FilePath     string
	Entries      []parser.LogEntry
	TotalLines   int
	ValidEntries int
	ParseErrors  int
}

type FileProcessError struct {
	FilePath string
	Error    error
}

type ProcessResult struct {
	FileResults     []FileReadResult
	FileErrors      []FileProcessError
	TotalFiles      int
	ProcessedFiles  int
	FailedFiles     int
	TotalLines      int
	ValidEntries    int
	ParseErrors     int
	MergedEntries   int
	RequestGroups   int
	OrphanedRecords int
	FailedRequests  int
	Analysis        reporter.AnalysisResult
}

type Processor struct {
	parser parser.LogParser
}

type LogEntry = parser.LogEntry

type FailedRequest struct {
	RequestID    string
	FirstFailure LogEntry
}

type fileReadError struct {
	filePath string
	err      error
}

func (e fileReadError) Error() string {
	return fmt.Sprintf("read file %q: %v", e.filePath, e.err)
}

func (e fileReadError) Unwrap() error {
	return e.err
}

func New() Processor {
	return Processor{parser: parser.LogParser{}}
}

func (p Processor) ProcessFiles(paths []string) ProcessResult {
	return p.ProcessFilesWithContext(context.Background(), paths)
}

func (p Processor) ProcessFilesWithContext(ctx context.Context, paths []string) ProcessResult {
	return p.processFilesWithWorkerPool(ctx, paths, defaultWorkerCount(len(paths)))
}

func (p Processor) processFilesWithWorkerPool(ctx context.Context, paths []string, numWorkers int) ProcessResult {
	result := ProcessResult{TotalFiles: len(paths)}
	allEntries := make([]parser.LogEntry, 0)
	if len(paths) == 0 {
		p.fillCorrelationData(allEntries, &result)
		return result
	}

	jobs := make(chan string)
	results := make(chan FileReadResult)
	errs := make(chan error)
	var wg sync.WaitGroup

	for workerID := 0; workerID < numWorkers; workerID++ {
		wg.Add(1)
		go p.fileResultWorker(ctx, jobs, results, errs, &wg)
	}

	go func() {
		for _, path := range paths {
			select {
			case <-ctx.Done():
				close(jobs)
				return
			case jobs <- path:
			}
		}
		close(jobs)
	}()
	go func() {
		wg.Wait()
		close(results)
		close(errs)
	}()

	resultsOpen := true
	errsOpen := true
	for resultsOpen || errsOpen {
		select {
		case fileResult, ok := <-results:
			if !ok {
				resultsOpen = false
				continue
			}

			result.ProcessedFiles++
			result.FileResults = append(result.FileResults, fileResult)
			result.TotalLines += fileResult.TotalLines
			result.ValidEntries += fileResult.ValidEntries
			result.ParseErrors += fileResult.ParseErrors
			allEntries = append(allEntries, fileResult.Entries...)
		case err, ok := <-errs:
			if !ok {
				errsOpen = false
				continue
			}
			if err == nil {
				continue
			}

			result.FailedFiles++
			result.FileErrors = append(result.FileErrors, toFileProcessError(err))
		}
	}

	p.fillCorrelationData(allEntries, &result)
	return result
}

func (p Processor) fileResultWorker(
	ctx context.Context,
	jobs <-chan string,
	results chan<- FileReadResult,
	errs chan<- error,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case path, ok := <-jobs:
			if !ok {
				return
			}

			fileResult, err := p.ReadLogFile(path)
			if err != nil {
				errs <- fileReadError{filePath: path, err: err}
				continue
			}

			results <- fileResult
		}
	}
}

func ProcessFilesConcurrently(
	ctx context.Context,
	filePaths []string,
	numWorkers int,
) ([]LogEntry, error) {
	if numWorkers <= 0 {
		return nil, fmt.Errorf("numWorkers must be greater than 0")
	}
	if len(filePaths) == 0 {
		return []LogEntry{}, nil
	}

	jobs := make(chan string)
	results := make(chan []LogEntry)
	errs := make(chan error)
	var wg sync.WaitGroup
	processor := New()

	for workerID := 0; workerID < numWorkers; workerID++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			processor.fileWorker(ctx, jobs, results, errs)
		}()
	}

	go func() {
		for _, path := range filePaths {
			select {
			case <-ctx.Done():
				close(jobs)
				return
			case jobs <- path:
			}
		}
		close(jobs)
	}()
	go func() {
		wg.Wait()
		close(results)
		close(errs)
	}()

	merged := make([]LogEntry, 0)
	collectedErrors := make([]error, 0)
	resultsOpen := true
	errsOpen := true
	for resultsOpen || errsOpen {
		select {
		case entries, ok := <-results:
			if !ok {
				resultsOpen = false
				continue
			}
			merged = append(merged, entries...)
		case err, ok := <-errs:
			if !ok {
				errsOpen = false
				continue
			}
			if err != nil {
				collectedErrors = append(collectedErrors, err)
			}
		}
	}

	if ctx.Err() != nil {
		collectedErrors = append(collectedErrors, ctx.Err())
	}
	return merged, errors.Join(collectedErrors...)
}

func (p Processor) fileWorker(
	ctx context.Context,
	jobs <-chan string,
	results chan<- []LogEntry,
	errs chan<- error,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case path, ok := <-jobs:
			if !ok {
				return
			}

			fileResult, err := p.ReadLogFile(path)
			if err != nil {
				errs <- fileReadError{filePath: path, err: err}
				continue
			}

			results <- fileResult.Entries
		}
	}
}

func defaultWorkerCount(totalFiles int) int {
	if totalFiles <= 0 {
		return 1
	}

	numWorkers := runtime.NumCPU()
	if numWorkers > totalFiles {
		return totalFiles
	}
	if numWorkers <= 0 {
		return 1
	}

	return numWorkers
}

func toFileProcessError(err error) FileProcessError {
	var readErr fileReadError
	if errors.As(err, &readErr) {
		return FileProcessError{FilePath: readErr.filePath, Error: readErr.err}
	}

	return FileProcessError{FilePath: "", Error: err}
}

func (p Processor) ReadLogFile(path string) (FileReadResult, error) {
	file, err := os.Open(path)
	if err != nil {
		return FileReadResult{}, fmt.Errorf("open file %q: %w", path, err)
	}
	defer file.Close()

	result := FileReadResult{FilePath: path}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		result.TotalLines++
		entry, err := p.parser.ParseLine(scanner.Text())
		if err != nil {
			result.ParseErrors++
			continue
		}

		result.Entries = append(result.Entries, entry)
		result.ValidEntries++
	}

	if err := scanner.Err(); err != nil {
		return FileReadResult{}, fmt.Errorf("scan file %q: %w", path, err)
	}

	return result, nil
}

func (p Processor) fillCorrelationData(entries []parser.LogEntry, result *ProcessResult) {
	grouped, orphanedCount := p.CorrelateRequests(entries)
	requestIDs := sortedRequestIDs(grouped)
	failedRequests := p.DetectFailedRequests(grouped)
	failedByRequestID := mapFailedByRequestID(failedRequests)

	report := reporter.AnalysisResult{FailedRequests: make([]reporter.FailedRequestReport, 0)}
	for _, requestID := range requestIDs {
		sortedTimeline := SortTimelineByTimestamp(grouped[requestID])
		firstFailure, ok := failedByRequestID[requestID]
		if !ok {
			continue
		}

		_ = firstFailure
		report.FailedRequests = append(report.FailedRequests, reporter.FailedRequestReport{
			RequestID: requestID,
			Timeline:  formatTimeline(sortedTimeline),
		})
	}

	result.MergedEntries = len(entries)
	result.RequestGroups = len(requestIDs)
	result.OrphanedRecords = orphanedCount
	result.FailedRequests = len(report.FailedRequests)
	result.Analysis = report
}

func (p Processor) CorrelateRequests(entries []parser.LogEntry) (map[string][]parser.LogEntry, int) {
	grouped := make(map[string][]parser.LogEntry)
	orphanedCount := 0

	for _, entry := range entries {
		if entry.RequestID == "" {
			orphanedCount++
			continue
		}

		grouped[entry.RequestID] = append(grouped[entry.RequestID], entry)
	}

	return grouped, orphanedCount
}

func (p Processor) DetectFailedRequests(grouped map[string][]parser.LogEntry) []FailedRequest {
	requestIDs := sortedRequestIDs(grouped)
	failed := make([]FailedRequest, 0, len(requestIDs))

	for _, requestID := range requestIDs {
		sortedTimeline := SortTimelineByTimestamp(grouped[requestID])
		firstFailure, ok := p.FindFirstFailure(sortedTimeline)
		if !ok {
			continue
		}

		failed = append(failed, FailedRequest{RequestID: requestID, FirstFailure: firstFailure})
	}

	return failed
}

func sortedRequestIDs(grouped map[string][]parser.LogEntry) []string {
	requestIDs := make([]string, 0, len(grouped))
	for requestID := range grouped {
		requestIDs = append(requestIDs, requestID)
	}

	sort.Strings(requestIDs)
	return requestIDs
}

func SortTimelineByTimestamp(entries []parser.LogEntry) []parser.LogEntry {
	sorted := append([]parser.LogEntry(nil), entries...)
	sort.SliceStable(sorted, func(i int, j int) bool {
		return compareEntries(sorted[i], sorted[j]) < 0
	})
	return sorted
}

func compareEntries(a parser.LogEntry, b parser.LogEntry) int {
	if a.Timestamp.IsZero() && !b.Timestamp.IsZero() {
		return 1
	}
	if !a.Timestamp.IsZero() && b.Timestamp.IsZero() {
		return -1
	}
	if a.Timestamp.Before(b.Timestamp) {
		return -1
	}
	if b.Timestamp.Before(a.Timestamp) {
		return 1
	}

	return strings.Compare(tieBreakerKey(a), tieBreakerKey(b))
}

func tieBreakerKey(entry parser.LogEntry) string {
	return entry.Level + "|" + entry.Service + "|" + entry.Message + "|" + entry.RequestID + "|" + entry.UserID
}

func (p Processor) FindFirstFailure(entries []parser.LogEntry) (parser.LogEntry, bool) {
	for _, entry := range entries {
		if entry.Level == "ERROR" || entry.Level == "WARN" {
			return entry, true
		}
	}

	return parser.LogEntry{}, false
}

func mapFailedByRequestID(failed []FailedRequest) map[string]parser.LogEntry {
	index := make(map[string]parser.LogEntry, len(failed))
	for _, request := range failed {
		index[request.RequestID] = request.FirstFailure
	}

	return index
}

func formatTimeline(entries []parser.LogEntry) []string {
	timeline := make([]string, 0, len(entries))
	for _, entry := range entries {
		timeline = append(timeline, fmt.Sprintf(
			"%s [%s] %s: %s",
			entry.Timestamp.Format("2006-01-02T15:04:05.999999999Z07:00"),
			entry.Level,
			entry.Service,
			entry.Message,
		))
	}

	return timeline
}

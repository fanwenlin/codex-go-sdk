package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fanwenlin/codex-go-sdk/types"
)

type rpcError struct {
	Code    int             `json:"code,omitempty"`
	Message string          `json:"message,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type rpcEnvelope struct {
	ID     *int64          `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *rpcError       `json:"error,omitempty"`
}

type appEvent struct {
	Method string
	Params json.RawMessage
}

// AppServerExec handles execution of the Codex app server.
type AppServerExec struct {
	executablePath string
	args           []string
	envOverride    map[string]string
	clientInfo     types.ClientInfo
	baseURL        string
	apiKey         string

	verbose       bool
	verboseWriter io.Writer

	startOnce sync.Once
	startErr  error

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser

	writeMu sync.Mutex

	nextID    int64
	pending   map[int64]chan rpcEnvelope
	pendingMu sync.Mutex

	subsMu sync.RWMutex
	subs   map[chan appEvent]struct{}

	knownThreadsMu sync.Mutex
	knownThreads   map[string]struct{}
}

// NewAppServerExec creates a new AppServerExec instance.
func NewAppServerExec(executablePath string, args []string, envOverride map[string]string, clientInfo types.ClientInfo, baseURL string, apiKey string) *AppServerExec {
	if executablePath == "" {
		executablePath = findCodexPath()
	}
	if len(args) == 0 {
		args = []string{"app-server"}
	}
	if clientInfo.Name == "" {
		clientInfo.Name = "codex-go-sdk"
	}
	if clientInfo.Version == "" {
		clientInfo.Version = "dev"
	}
	return &AppServerExec{
		executablePath: executablePath,
		args:           args,
		envOverride:    envOverride,
		clientInfo:     clientInfo,
		baseURL:        baseURL,
		apiKey:         apiKey,
		pending:        make(map[int64]chan rpcEnvelope),
		subs:           make(map[chan appEvent]struct{}),
		knownThreads:   make(map[string]struct{}),
	}
}

// EnableVerbose enables debug logging for the app server exec.
func (a *AppServerExec) EnableVerbose(writer io.Writer) {
	a.verbose = true
	if writer != nil {
		a.verboseWriter = writer
	} else {
		a.verboseWriter = os.Stderr
	}
}

func (a *AppServerExec) logf(format string, args ...interface{}) {
	if !a.verbose {
		return
	}
	if a.verboseWriter == nil {
		a.verboseWriter = os.Stderr
	}
	fmt.Fprintf(a.verboseWriter, format+"\n", args...)
}

func (a *AppServerExec) ensureStarted() error {
	a.startOnce.Do(func() {
		a.startErr = a.start()
	})
	return a.startErr
}

func (a *AppServerExec) start() error {
	cmd := exec.Command(a.executablePath, a.args...)

	// Set up environment
	env := os.Environ()
	if a.envOverride != nil {
		env = []string{}
		for k, v := range a.envOverride {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
	}
	foundOriginator := false
	for _, e := range env {
		if strings.HasPrefix(e, envInternalOriginatorOverrideKey+"=") {
			foundOriginator = true
			break
		}
	}
	if !foundOriginator {
		env = append(env, envInternalOriginatorOverrideKey+"=codex_sdk_go")
	}
	if a.baseURL != "" {
		env = append(env, fmt.Sprintf("%s=%s", envBaseURLKey, a.baseURL))
	}
	if a.apiKey != "" {
		env = append(env, fmt.Sprintf("%s=%s", envCodexAPIEnvVar, a.apiKey))
	}
	cmd.Env = env

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create app server stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create app server stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create app server stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start app server: %w", err)
	}

	a.cmd = cmd
	a.stdin = stdin
	a.stdout = stdout

	go a.readStdout()
	go a.readStderr(stderr)

	// Initialize protocol.
	ctx, cancel := context.WithTimeout(context.Background(), defaultInitTimeout)
	defer cancel()
	if err := a.initialize(ctx); err != nil {
		return err
	}
	return nil
}

func (a *AppServerExec) readStdout() {
	reader := bufio.NewReader(a.stdout)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			line = strings.TrimRight(line, "\r\n")
			if line != "" {
				a.handleLine(line)
			}
		}
		if err != nil {
			if err != io.EOF {
				a.logf("app server stdout error: %v", err)
			}
			return
		}
	}
}

func (a *AppServerExec) readStderr(stderr io.Reader) {
	reader := bufio.NewReader(stderr)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			a.logf("app server stderr: %s", strings.TrimRight(line, "\r\n"))
		}
		if err != nil {
			if err != io.EOF {
				a.logf("app server stderr error: %v", err)
			}
			return
		}
	}
}

func (a *AppServerExec) handleLine(line string) {
	var envelope rpcEnvelope
	if err := json.Unmarshal([]byte(line), &envelope); err != nil {
		a.logf("app server: failed to parse line: %v", err)
		return
	}
	if envelope.ID != nil {
		a.pendingMu.Lock()
		ch := a.pending[*envelope.ID]
		a.pendingMu.Unlock()
		if ch != nil {
			ch <- envelope
		}
		return
	}
	if envelope.Method != "" {
		a.dispatchEvent(appEvent{Method: envelope.Method, Params: envelope.Params})
	}
}

func (a *AppServerExec) dispatchEvent(event appEvent) {
	a.subsMu.RLock()
	for ch := range a.subs {
		select {
		case ch <- event:
		default:
			// Drop if the subscriber is too slow.
		}
	}
	a.subsMu.RUnlock()
}

func (a *AppServerExec) subscribe() chan appEvent {
	ch := make(chan appEvent, 256)
	a.subsMu.Lock()
	a.subs[ch] = struct{}{}
	a.subsMu.Unlock()
	return ch
}

func (a *AppServerExec) unsubscribe(ch chan appEvent) {
	a.subsMu.Lock()
	if _, ok := a.subs[ch]; ok {
		delete(a.subs, ch)
		close(ch)
	}
	a.subsMu.Unlock()
}

func (a *AppServerExec) call(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	id := atomic.AddInt64(&a.nextID, 1)
	respCh := make(chan rpcEnvelope, 1)

	a.pendingMu.Lock()
	a.pending[id] = respCh
	a.pendingMu.Unlock()

	if err := a.sendRequest(id, method, params); err != nil {
		a.pendingMu.Lock()
		delete(a.pending, id)
		a.pendingMu.Unlock()
		return nil, err
	}

	select {
	case resp := <-respCh:
		a.pendingMu.Lock()
		delete(a.pending, id)
		a.pendingMu.Unlock()
		if resp.Error != nil {
			return nil, fmt.Errorf("app server error (%d): %s", resp.Error.Code, resp.Error.Message)
		}
		return resp.Result, nil
	case <-ctx.Done():
		a.pendingMu.Lock()
		delete(a.pending, id)
		a.pendingMu.Unlock()
		return nil, ctx.Err()
	}
}

func (a *AppServerExec) notify(method string, params interface{}) error {
	return a.sendRequest(0, method, params)
}

func (a *AppServerExec) sendRequest(id int64, method string, params interface{}) error {
	payload := map[string]interface{}{
		"method": method,
	}
	if id != 0 {
		payload["id"] = id
	}
	if params != nil {
		payload["params"] = params
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	a.writeMu.Lock()
	defer a.writeMu.Unlock()
	if _, err := a.stdin.Write(append(data, '\n')); err != nil {
		return err
	}
	return nil
}

func (a *AppServerExec) initialize(ctx context.Context) error {
	params := map[string]interface{}{
		"clientInfo": map[string]string{
			"name":    a.clientInfo.Name,
			"version": a.clientInfo.Version,
		},
	}
	if _, err := a.call(ctx, "initialize", params); err != nil {
		return err
	}
	return a.notify("initialized", nil)
}

// Run executes the app server turn and returns a channel of JSONL event lines.
func (a *AppServerExec) Run(args CodexExecArgs) <-chan ExecResult {
	output := make(chan ExecResult)

	go func() {
		defer close(output)

		if err := a.ensureStarted(); err != nil {
			output <- ExecResult{Error: err}
			return
		}

		ctx := args.Context
		if ctx == nil {
			ctx = context.Background()
		}

		threadID, isNewThread, err := a.ensureThread(ctx, args.ThreadId, args.Model)
		if err != nil {
			output <- ExecResult{Error: err}
			return
		}

		if isNewThread {
			threadStarted := map[string]interface{}{
				"type":      "thread.started",
				"thread_id": threadID,
			}
			if line, err := json.Marshal(threadStarted); err == nil {
				output <- ExecResult{Line: string(line)}
			}
		}

		inputItems, err := buildInputItems(args)
		if err != nil {
			output <- ExecResult{Error: err}
			return
		}

		turnParams := map[string]interface{}{
			"threadId": threadID,
			"input":    inputItems,
			"stream":   true,
		}
		if args.Model != "" {
			turnParams["model"] = args.Model
		}
		if args.ModelReasoningEffort != "" {
			turnParams["effort"] = args.ModelReasoningEffort
		}
		if args.WorkingDirectory != "" {
			turnParams["cwd"] = args.WorkingDirectory
		}
		if sandbox := buildSandboxPolicy(args); sandbox != nil {
			turnParams["sandboxPolicy"] = sandbox
		}
		if approval := mapApprovalPolicy(args.ApprovalPolicy); approval != "" {
			turnParams["approvalPolicy"] = approval
		}
		if schema, err := loadOutputSchema(args.OutputSchemaFile); err != nil {
			output <- ExecResult{Error: err}
			return
		} else if schema != nil {
			turnParams["outputSchema"] = schema
		}

		result, err := a.call(ctx, "turn/start", turnParams)
		if err != nil {
			output <- ExecResult{Error: err}
			return
		}

		turnID := extractTurnID(result)
		if turnID == "" {
			// If the server did not return a turn id, rely on events to detect completion.
			a.logf("app server: missing turn id in response")
		}

		sub := a.subscribe()
		defer a.unsubscribe(sub)

		state := &turnState{
			items: make(map[string]map[string]interface{}),
		}

		for {
			select {
			case <-ctx.Done():
				output <- ExecResult{Error: ctx.Err()}
				return
			case event, ok := <-sub:
				if !ok {
					return
				}
				if !eventMatchesTurn(event, threadID, turnID) {
					continue
				}
				if args.ApprovalHandler != nil {
					if event.Method == "item/commandExecution/approvalRequested" || event.Method == "item/fileChange/approvalRequested" {
						a.submitApproval(ctx, event, args.ApprovalHandler)
					}
				}
				line, done, err := appEventToLegacyLine(event, state)
				if err != nil {
					output <- ExecResult{Error: err}
					return
				}
				if line != "" {
					output <- ExecResult{Line: line}
				}
				if done {
					return
				}
			}
		}
	}()

	return output
}

const defaultInitTimeout = 10 * time.Second

func (a *AppServerExec) ensureThread(ctx context.Context, requested *string, model string) (string, bool, error) {
	if requested == nil || *requested == "" {
		params := map[string]interface{}{}
		if model != "" {
			params["model"] = model
		}
		result, err := a.call(ctx, "thread/start", params)
		if err != nil {
			return "", false, err
		}
		threadID := extractThreadID(result)
		if threadID == "" {
			return "", false, fmt.Errorf("app server did not return thread id")
		}
		a.knownThreadsMu.Lock()
		a.knownThreads[threadID] = struct{}{}
		a.knownThreadsMu.Unlock()
		return threadID, true, nil
	}

	threadID := *requested
	a.knownThreadsMu.Lock()
	_, known := a.knownThreads[threadID]
	a.knownThreadsMu.Unlock()
	if !known {
		_, err := a.call(ctx, "thread/resume", map[string]interface{}{
			"threadId": threadID,
		})
		if err != nil {
			return "", false, err
		}
		a.knownThreadsMu.Lock()
		a.knownThreads[threadID] = struct{}{}
		a.knownThreadsMu.Unlock()
	}
	return threadID, false, nil
}

type turnState struct {
	items map[string]map[string]interface{}
}

func appEventToLegacyLine(event appEvent, state *turnState) (string, bool, error) {
	method := event.Method
	switch method {
	case "item/agentMessage/delta":
		return applyTextDelta(event, state, "agentMessage", "text")
	case "item/reasoning/summaryTextDelta":
		return applyTextDelta(event, state, "reasoning", "text")
	case "item/commandExecution/outputDelta":
		return applyTextDelta(event, state, "commandExecution", "aggregatedOutput")
	case "item/fileChange/outputDelta":
		return applyTextDelta(event, state, "fileChange", "output")
	}

	payload := map[string]interface{}{}
	if len(event.Params) > 0 {
		if err := json.Unmarshal(event.Params, &payload); err != nil {
			return "", false, err
		}
	}
	payload["type"] = strings.ReplaceAll(method, "/", ".")

	if item, ok := payload["item"].(map[string]interface{}); ok {
		if id, ok := item["id"].(string); ok {
			state.items[id] = item
		}
	}

	line, err := json.Marshal(payload)
	if err != nil {
		return "", false, err
	}
	done := method == "turn/completed" || method == "turn/failed"
	return string(line), done, nil
}

func applyTextDelta(event appEvent, state *turnState, itemType string, field string) (string, bool, error) {
	var params struct {
		ItemID string `json:"itemId"`
		Delta  string `json:"delta"`
	}
	if err := json.Unmarshal(event.Params, &params); err != nil {
		return "", false, err
	}
	if params.ItemID == "" {
		return "", false, nil
	}
	item, ok := state.items[params.ItemID]
	if !ok {
		item = map[string]interface{}{
			"id":   params.ItemID,
			"type": itemType,
		}
		state.items[params.ItemID] = item
	}
	if params.Delta != "" {
		if existing, ok := item[field].(string); ok {
			item[field] = existing + params.Delta
		} else {
			item[field] = params.Delta
		}
	}
	payload := map[string]interface{}{
		"type": "item.updated",
		"item": item,
	}
	line, err := json.Marshal(payload)
	if err != nil {
		return "", false, err
	}
	return string(line), false, nil
}

func eventMatchesTurn(event appEvent, threadID string, turnID string) bool {
	if threadID == "" && turnID == "" {
		return true
	}
	var meta struct {
		ThreadID string `json:"threadId"`
		TurnID   string `json:"turnId"`
		Turn     *struct {
			ID string `json:"id"`
		} `json:"turn"`
	}
	if err := json.Unmarshal(event.Params, &meta); err != nil {
		return false
	}
	if meta.TurnID == "" && meta.Turn != nil {
		meta.TurnID = meta.Turn.ID
	}
	if turnID != "" && meta.TurnID != "" {
		return meta.TurnID == turnID
	}
	if threadID != "" && meta.ThreadID != "" {
		return meta.ThreadID == threadID
	}
	return false
}

func (a *AppServerExec) submitApproval(ctx context.Context, event appEvent, handler types.ApprovalHandler) {
	var params struct {
		ThreadID string `json:"threadId"`
		ItemID   string `json:"itemId"`
		Item     *struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"item"`
	}
	if err := json.Unmarshal(event.Params, &params); err != nil {
		a.logf("app server: failed to parse approval request: %v", err)
		return
	}
	itemID := params.ItemID
	itemType := ""
	if params.Item != nil {
		if itemID == "" {
			itemID = params.Item.ID
		}
		itemType = params.Item.Type
	}
	if itemID == "" {
		return
	}
	decision, err := handler(types.ApprovalRequest{
		ItemID:   itemID,
		ItemType: itemType,
	})
	if err != nil {
		a.logf("app server: approval handler error: %v", err)
		return
	}
	if decision == "" {
		return
	}
	payload := map[string]interface{}{
		"itemId":   itemID,
		"decision": string(decision),
	}
	if params.ThreadID != "" {
		payload["threadId"] = params.ThreadID
	}
	if _, err := a.call(ctx, "approval/submit", payload); err != nil {
		a.logf("app server: approval submit error: %v", err)
	}
}

func buildInputItems(args CodexExecArgs) ([]map[string]interface{}, error) {
	var items []map[string]interface{}

	if len(args.InputItems) == 0 && args.Input != "" {
		args.InputItems = []types.UserInput{types.NewTextInput(args.Input)}
	}

	for _, item := range args.InputItems {
		switch item.Type {
		case "text":
			if item.Text == "" {
				continue
			}
			items = append(items, map[string]interface{}{
				"type": "text",
				"text": item.Text,
			})
		case "local_image", "localImage":
			if item.Path == "" {
				continue
			}
			items = append(items, map[string]interface{}{
				"type": "localImage",
				"path": item.Path,
			})
		case "image":
			url := item.URL
			if url == "" {
				url = item.Path
			}
			if url == "" {
				continue
			}
			items = append(items, map[string]interface{}{
				"type": "image",
				"url":  url,
			})
		case "skill":
			name := item.Name
			if name == "" {
				name = item.Text
			}
			if name == "" {
				continue
			}
			items = append(items, map[string]interface{}{
				"type": "skill",
				"name": name,
			})
		case "mention":
			text := item.Text
			if text == "" {
				text = item.Name
			}
			if text == "" {
				continue
			}
			items = append(items, map[string]interface{}{
				"type": "mention",
				"text": text,
			})
		}
	}

	for _, image := range args.Images {
		if image == "" {
			continue
		}
		items = append(items, map[string]interface{}{
			"type": "localImage",
			"path": image,
		})
	}

	return items, nil
}

func buildSandboxPolicy(args CodexExecArgs) map[string]interface{} {
	policy := map[string]interface{}{}
	switch args.SandboxMode {
	case "read-only":
		policy["type"] = "readOnly"
	case "workspace-write":
		policy["type"] = "workspaceWrite"
	case "danger-full-access":
		policy["type"] = "dangerFullAccess"
	}
	if len(policy) == 0 {
		return nil
	}
	networkAccess := "disabled"
	if args.NetworkAccessEnabled {
		networkAccess = "enabled"
	}
	policy["networkAccess"] = networkAccess
	if args.WorkingDirectory != "" || len(args.AdditionalDirectories) > 0 {
		roots := []string{}
		if args.WorkingDirectory != "" {
			roots = append(roots, args.WorkingDirectory)
		}
		for _, dir := range args.AdditionalDirectories {
			if dir == "" {
				continue
			}
			roots = append(roots, dir)
		}
		policy["writableRoots"] = roots
	}
	return policy
}

func mapApprovalPolicy(policy string) string {
	switch policy {
	case "on-request":
		return "onRequest"
	case "on-failure":
		return "onFailure"
	case "untrusted":
		return "unlessTrusted"
	default:
		return policy
	}
}

func loadOutputSchema(path string) (interface{}, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var schema interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, err
	}
	return schema, nil
}

func extractThreadID(result json.RawMessage) string {
	var payload struct {
		Thread *struct {
			ID string `json:"id"`
		} `json:"thread"`
		ThreadID string `json:"threadId"`
	}
	if err := json.Unmarshal(result, &payload); err != nil {
		return ""
	}
	if payload.Thread != nil && payload.Thread.ID != "" {
		return payload.Thread.ID
	}
	return payload.ThreadID
}

func extractTurnID(result json.RawMessage) string {
	var payload struct {
		Turn *struct {
			ID string `json:"id"`
		} `json:"turn"`
		TurnID string `json:"turnId"`
	}
	if err := json.Unmarshal(result, &payload); err != nil {
		return ""
	}
	if payload.Turn != nil && payload.Turn.ID != "" {
		return payload.Turn.ID
	}
	return payload.TurnID
}

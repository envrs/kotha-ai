package agent

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/kothagpt/kotha/internal/config"
	"github.com/kothagpt/kotha/internal/llm/models"
	"github.com/kothagpt/kotha/internal/llm/provider"
	"github.com/kothagpt/kotha/internal/llm/tools"
	"github.com/kothagpt/kotha/internal/message"
	"github.com/kothagpt/kotha/internal/observability"
	"github.com/kothagpt/kotha/internal/plugin"
	"github.com/stretchr/testify/require"
)

// Test agent core functionality by constructing minimal agents directly.

func TestAgent_ExtractFilePath(t *testing.T) {
	require.Equal(t, "foo.go", extractFilePath(`{"file_path":"foo.go","old_string":"a"}`))
	require.Equal(t, "bar.py", extractFilePath(`{"file_path":"bar.py","new_string":"b"}`))
	require.Equal(t, "", extractFilePath(`{"url":"http://example.com"}`))
	require.Equal(t, "", extractFilePath(""))
	require.Equal(t, "", extractFilePath(`{"command":"ls"}`))
}

func TestAgent_GenerateAdmissionID(t *testing.T) {
	id1 := generateAdmissionID("sess-1", "hello world")
	id2 := generateAdmissionID("sess-1", "hello world")
	id3 := generateAdmissionID("sess-2", "hello world")

	require.Equal(t, id1, id2) // Same session + content = same ID
	require.NotEqual(t, id1, id3) // Different session = different ID
	require.True(t, len(id1) > 0)
}

func TestAgent_GenerateFileDraftID(t *testing.T) {
	id1 := generateFileDraftID()
	id2 := generateFileDraftID()

	require.NotEqual(t, id1, id2) // Should be unique
	require.True(t, len(id1) > 0)
}

func TestAgent_Err(t *testing.T) {
	ag := &agent{}
	event := ag.err(errors.New("test error"))
	require.Equal(t, AgentEventTypeError, event.Type)
	require.Equal(t, "test error", event.Error.Error())
}

func TestAgent_ErrRetryRequested(t *testing.T) {
	ag := &agent{logger: &fakeLogger{}}
	err := ag.processEvent(context.Background(), "test", &message.Message{}, provider.ProviderEvent{
		Type:  provider.EventError,
		Error: ErrRetryRequested,
	})
	require.ErrorIs(t, err, ErrRetryRequested)
}

func TestAgent_RetryHookRegistry(t *testing.T) {
	registry := plugin.NewRetryHookRegistry()
	registry.RegisterGlobal(func(ctx context.Context, rc *plugin.RetryContext) (plugin.RetryDecision, error) {
		return plugin.RetryDecisionRetry, nil
	})

	ag := &agent{
		retryHookRegistry: registry,
		provider:          newFakeProvider("test-model"),
		logger:            &fakeLogger{},
	}
	err := ag.processEvent(context.Background(), "test", &message.Message{}, provider.ProviderEvent{
		Type:  provider.EventError,
		Error: fmt.Errorf("retry requested by hook: %w", ErrRetryRequested),
	})
	require.ErrorIs(t, err, ErrRetryRequested)
}

func TestAgent_Meter(t *testing.T) {
	ag := &agent{meter: observability.NewMeter()}
	meter := ag.Meter()
	require.NotNil(t, meter)
}

func TestAgent_Model(t *testing.T) {
	prov := newFakeProvider("test-model")
	ag := &agent{provider: prov}
	require.Equal(t, models.ModelID("test-model"), ag.Model().ID)
}

func TestAgent_Cancel(t *testing.T) {
	ag := &agent{}
	ag.Cancel("test-session")
	// Should not panic
}

func TestAgent_IsBusy(t *testing.T) {
	ag := &agent{}
	require.False(t, ag.IsBusy())
	require.False(t, ag.IsSessionBusy("test-session"))
}


func TestAgent_Update(t *testing.T) {
	prov := newFakeProvider("test-model")
	ag := &agent{
		provider:         prov,
		titleProvider:    newFakeProvider("test-model"),
		summarizeProvider: newFakeProvider("test-model"),
		logger:            &fakeLogger{},
		configProvider:    &fakeConfigProvider{},
	}
	model, err := ag.Update(config.AgentCoder, models.ModelID("new-model"))
	require.Error(t, err) // Will fail because we don't have real configProvider
	require.Equal(t, models.Model{}, model)
}

func TestAgent_Meter_Nil(t *testing.T) {
	ag := &agent{}
	require.Nil(t, ag.Meter())
}

// fakeProvider is a minimal provider for testing.
type fakeProvider struct {
	model models.Model
}

func newFakeProvider(modelID string) *fakeProvider {
	return &fakeProvider{
		model: models.Model{ID: models.ModelID(modelID)},
	}
}

func (f *fakeProvider) SendMessages(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (*provider.ProviderResponse, error) {
	return &provider.ProviderResponse{
		Content:      "test response",
		FinishReason: message.FinishReasonEndTurn,
	}, nil
}

func (f *fakeProvider) StreamResponse(ctx context.Context, messages []message.Message, tools []tools.BaseTool) <-chan provider.ProviderEvent {
	ch := make(chan provider.ProviderEvent, 2)
	go func() {
		defer close(ch)
		ch <- provider.ProviderEvent{Type: provider.EventContentDelta, Content: "test"}
		ch <- provider.ProviderEvent{
			Type: provider.EventComplete,
			Response: &provider.ProviderResponse{
				Content:      "test",
				FinishReason: message.FinishReasonEndTurn,
			},
		}
	}()
	return ch
}

func (f *fakeProvider) Model() models.Model {
	return f.model
}

// fakeLogger is a minimal logger for testing.
type fakeLogger struct{}

func (f *fakeLogger) Info(msg string, args ...interface{}) {}
func (f *fakeLogger) Debug(msg string, args ...interface{}) {}
func (f *fakeLogger) Warn(msg string, args ...interface{}) {}
func (f *fakeLogger) Error(msg string, args ...interface{}) {}
func (f *fakeLogger) InfoPersist(msg string, args ...interface{}) {}
func (f *fakeLogger) DebugPersist(msg string, args ...interface{}) {}
func (f *fakeLogger) WarnPersist(msg string, args ...interface{}) {}
func (f *fakeLogger) ErrorPersist(msg string, args ...interface{}) {}
func (f *fakeLogger) RecoverPanic(name string, cleanup func()) {}
func (f *fakeLogger) AppendToSessionLogFile(sessionId string, filename string, content string) string { return "" }
func (f *fakeLogger) WriteRequestMessageJson(sessionId string, requestSeqId int, message interface{}) string { return "" }
func (f *fakeLogger) WriteRequestMessage(sessionId string, requestSeqId int, message string) string { return "" }
func (f *fakeLogger) AppendToStreamSessionLogJson(sessionId string, requestSeqId int, jsonableChunk interface{}) string { return "" }
func (f *fakeLogger) AppendToStreamSessionLog(sessionId string, requestSeqId int, chunk string) string { return "" }
func (f *fakeLogger) WriteChatResponseJson(sessionId string, requestSeqId int, response interface{}) string { return "" }
func (f *fakeLogger) WriteToolResultsJson(sessionId string, requestSeqId int, toolResults interface{}) string { return "" }

// fakeConfigProvider is a minimal config provider for testing.
type fakeConfigProvider struct{}

func (f *fakeConfigProvider) Get() *config.Config {
	return &config.Config{}
}

func (f *fakeConfigProvider) WorkingDirectory() string {
	return "/tmp"
}

func (f *fakeConfigProvider) UpdateAgentModel(agentName config.AgentName, modelID models.ModelID) error {
	return nil
}

func (f *fakeConfigProvider) UpdateTheme(themeName string) error {
	return nil
}

func (f *fakeConfigProvider) SetProvider(provider models.ModelProvider, enabled bool, apiKey string) error {
	return nil
}

var _ = time.Now
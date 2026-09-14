package log_test

import (
	"context"
	"testing"

	corelog "github.com/jasonmiller-cc/parameters-core/pkg/log"
)

func TestNew_DoesNotPanic(t *testing.T) {
	l := corelog.New(corelog.LevelInfo, corelog.FormatJSON)
	l.Info("hello")

	l2 := corelog.New(corelog.LevelDebug, corelog.FormatText)
	l2.Debug("hello text")
}

func TestWithContext_FromContext_RoundTrip(t *testing.T) {
	l := corelog.New(corelog.LevelInfo, corelog.FormatJSON).With("service", "test")
	ctx := corelog.WithContext(context.Background(), l)

	got := corelog.FromContext(ctx)
	if got != l {
		t.Error("FromContext did not return the logger stored by WithContext")
	}
}

func TestFromContext_FallsBackToDefault(t *testing.T) {
	got := corelog.FromContext(context.Background())
	if got == nil {
		t.Fatal("FromContext returned nil for empty context")
	}
	if got != corelog.Default() {
		t.Error("FromContext should fall back to the default logger")
	}
}

func TestService(t *testing.T) {
	l := corelog.Service("myservice")
	if l == nil {
		t.Fatal("Service() returned nil")
	}
}

func TestSetDefault(t *testing.T) {
	original := corelog.Default()
	t.Cleanup(func() { corelog.SetDefault(original) })

	custom := corelog.New(corelog.LevelWarn, corelog.FormatText)
	corelog.SetDefault(custom)

	if corelog.Default() != custom {
		t.Error("SetDefault did not update the package default")
	}
}

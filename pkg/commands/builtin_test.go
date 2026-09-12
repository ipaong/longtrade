package commands

import (
	"context"
	"strings"
	"testing"
)

func findDefinitionByName(t *testing.T, defs []Definition, name string) Definition {
	t.Helper()
	for _, def := range defs {
		if def.Name == name {
			return def
		}
	}
	t.Fatalf("missing /%s definition", name)
	return Definition{}
}

func TestBuiltinHelpHandler_ReturnsFormattedMessage(t *testing.T) {
	defs := BuiltinDefinitions()
	helpDef := findDefinitionByName(t, defs, "help")
	if helpDef.Handler == nil {
		t.Fatalf("/help handler should not be nil")
	}

	var reply string
	err := helpDef.Handler(context.Background(), Request{
		Text: "/help",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	}, nil)
	if err != nil {
		t.Fatalf("/help handler error: %v", err)
	}
	// Now uses auto-generated EffectiveUsage which includes agents
	if !strings.Contains(reply, "/show [model|channel|agents]") {
		t.Fatalf("/help reply missing /show usage, got %q", reply)
	}
	if !strings.Contains(reply, "/list [models|channels|agents]") {
		t.Fatalf("/help reply missing /list usage, got %q", reply)
	}
}

func TestBuiltinShowChannel_PreservesUserVisibleBehavior(t *testing.T) {
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), nil)

	cases := []string{"telegram", "whatsapp"}
	for _, channel := range cases {
		var reply string
		res := ex.Execute(context.Background(), Request{
			Channel: channel,
			Text:    "/show channel",
			Reply: func(text string) error {
				reply = text
				return nil
			},
		})
		if res.Outcome != OutcomeHandled {
			t.Fatalf("/show channel on %s: outcome=%v, want=%v", channel, res.Outcome, OutcomeHandled)
		}
		want := "Current Channel: " + channel
		if reply != want {
			t.Fatalf("/show channel reply=%q, want=%q", reply, want)
		}
	}
}

func TestBuiltinListChannels_UsesGetEnabledChannels(t *testing.T) {
	rt := &Runtime{
		GetEnabledChannels: func() []string {
			return []string{"telegram", "slack"}
		},
	}
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/list channels",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/list channels: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !strings.Contains(reply, "telegram") || !strings.Contains(reply, "slack") {
		t.Fatalf("/list channels reply=%q, want telegram and slack", reply)
	}
}

func TestBuiltinShowAgents_RestoresOldBehavior(t *testing.T) {
	rt := &Runtime{
		ListAgentIDs: func() []string {
			return []string{"default", "coder"}
		},
	}
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/show agents",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/show agents: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !strings.Contains(reply, "default") || !strings.Contains(reply, "coder") {
		t.Fatalf("/show agents reply=%q, want agent IDs", reply)
	}
}

func TestBuiltinListAgents_RestoresOldBehavior(t *testing.T) {
	rt := &Runtime{
		ListAgentIDs: func() []string {
			return []string{"default", "coder"}
		},
	}
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/list agents",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/list agents: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !strings.Contains(reply, "default") || !strings.Contains(reply, "coder") {
		t.Fatalf("/list agents reply=%q, want agent IDs", reply)
	}
}

func TestClearCommand_Success(t *testing.T) {
	rt := &Runtime{
		ClearHistory: func() error {
			return nil
		},
	}
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/clear",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/clear: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if reply != "Chat history cleared!" {
		t.Fatalf("/clear reply=%q, want %q", reply, "Chat history cleared!")
	}
}

func TestClearCommand_Error(t *testing.T) {
	rt := &Runtime{
		ClearHistory: func() error {
			return context.Canceled
		},
	}
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/clear",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/clear error: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !strings.Contains(reply, "Failed to clear") {
		t.Fatalf("/clear error reply=%q, want error message", reply)
	}
}

func TestClearCommand_NilDep(t *testing.T) {
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), &Runtime{})

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/clear",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/clear nil: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if reply != "Command unavailable in current context." {
		t.Fatalf("/clear nil reply=%q, want unavailable", reply)
	}
}

func TestContextCommand_Success(t *testing.T) {
	rt := &Runtime{
		GetContextStats: func() *ContextStats {
			return &ContextStats{
				UsedTokens:       1000,
				TotalTokens:      4096,
				CompressAtTokens: 3500,
				UsedPercent:      25,
				MessageCount:     10,
			}
		},
	}
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/context",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/context: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !strings.Contains(reply, "1000") {
		t.Fatalf("/context reply=%q, want token stats", reply)
	}
}

func TestContextCommand_NoSession(t *testing.T) {
	rt := &Runtime{
		GetContextStats: func() *ContextStats {
			return nil
		},
	}
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/context",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/context no session: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if reply != "No active session context." {
		t.Fatalf("/context no session reply=%q, want no session msg", reply)
	}
}

func TestContextCommand_NilDep(t *testing.T) {
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), &Runtime{})

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/context",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/context nil: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if reply != "Command unavailable in current context." {
		t.Fatalf("/context nil reply=%q, want unavailable", reply)
	}
}

func TestContextCommand_OverLimit(t *testing.T) {
	rt := &Runtime{
		GetContextStats: func() *ContextStats {
			return &ContextStats{
				UsedTokens:       5000,
				TotalTokens:      4096,
				CompressAtTokens: 3500,
				UsedPercent:      150,
				MessageCount:     20,
			}
		},
	}
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/context",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/context over limit: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	// Should show 0 remaining tokens (clamped from negative)
	if !strings.Contains(reply, "Remaining: ~0") {
		t.Fatalf("/context over limit reply=%q, want clamped remaining", reply)
	}
}

func TestListCommand_Models(t *testing.T) {
	rt := &Runtime{
		GetModelInfo: func() (string, string) {
			return "gpt-4", "openai"
		},
	}
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/list models",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/list models: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !strings.Contains(reply, "gpt-4") {
		t.Fatalf("/list models reply=%q, want model name", reply)
	}
}

func TestListCommand_Channels(t *testing.T) {
	rt := &Runtime{
		GetEnabledChannels: func() []string {
			return []string{"telegram", "discord"}
		},
	}
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/list channels",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/list channels: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !strings.Contains(reply, "telegram") || !strings.Contains(reply, "discord") {
		t.Fatalf("/list channels reply=%q, want channel names", reply)
	}
}

func TestListCommand_NoChannels(t *testing.T) {
	rt := &Runtime{
		GetEnabledChannels: func() []string {
			return []string{}
		},
	}
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/list channels",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/list channels empty: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if reply != "No channels enabled" {
		t.Fatalf("/list channels empty reply=%q, want no channels msg", reply)
	}
}

func TestListCommand_Agents(t *testing.T) {
	rt := &Runtime{
		ListAgentIDs: func() []string {
			return []string{"main", "research"}
		},
	}
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/list agents",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/list agents: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !strings.Contains(reply, "main") || !strings.Contains(reply, "research") {
		t.Fatalf("/list agents reply=%q, want agent IDs", reply)
	}
}

func TestListCommand_NoAgents(t *testing.T) {
	rt := &Runtime{
		ListAgentIDs: func() []string {
			return []string{}
		},
	}
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/list agents",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/list agents empty: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if reply != "No agents registered" {
		t.Fatalf("/list agents empty reply=%q, want no agents msg", reply)
	}
}

func TestListCommand_NilDep(t *testing.T) {
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), &Runtime{})

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/list models",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/list models nil: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if reply != "Command unavailable in current context." {
		t.Fatalf("/list models nil reply=%q, want unavailable", reply)
	}
}

func TestPairCommand(t *testing.T) {
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), nil)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/pair",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/pair: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if reply != "You are already authorized. ✅" {
		t.Fatalf("/pair reply=%q, want authorized message", reply)
	}
}

func TestShowCommand_Model(t *testing.T) {
	rt := &Runtime{
		GetModelInfo: func() (string, string) {
			return "claude-3-opus", "anthropic"
		},
	}
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/show model",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/show model: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !strings.Contains(reply, "claude-3-opus") {
		t.Fatalf("/show model reply=%q, want model name", reply)
	}
}

func TestShowCommand_Channel(t *testing.T) {
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), nil)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Channel: "slack",
		Text:    "/show channel",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/show channel: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if reply != "Current Channel: slack" {
		t.Fatalf("/show channel reply=%q, want channel name", reply)
	}
}

func TestShowCommand_NilDep(t *testing.T) {
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), &Runtime{})

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/show model",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/show model nil: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if reply != "Command unavailable in current context." {
		t.Fatalf("/show model nil reply=%q, want unavailable", reply)
	}
}

func TestStartCommand(t *testing.T) {
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), nil)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/start",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/start: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if reply != "Hello! I am KhunQuant 🦞" {
		t.Fatalf("/start reply=%q, want greeting message", reply)
	}
}

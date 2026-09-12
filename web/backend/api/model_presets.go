package api

import "github.com/cryptoquantumwave/khunquant/pkg/config"

// ─────────────────────────────────────────────────────────────────────────────
// Credential-card model presets
//
// UPDATE THIS FILE when new models are released (frontier models change ~monthly).
// Each provider block is independent — just edit labels and model IDs.
//
// ModelID format: {provider-prefix}/{api-model-id}
//   openai/       → OpenAI API (Codex CLI uses ChatGPT session auth)
//   anthropic/    → Anthropic API
//   antigravity/  → Google Antigravity (cloudcode-pa.googleapis.com)
//
// Last verified: 2026-06-28
// ─────────────────────────────────────────────────────────────────────────────

// oauthProviderModelPresets defines the clickable model chips shown on each
// credential card. Order matters — first entry is shown leftmost/topmost.
//
// Sources checked when last updated:
//
//	OpenAI:      developers.openai.com/codex/models
//	Anthropic:   platform.claude.com/docs/en/about-claude/models/overview
//	Antigravity: github.com/NoeFabris/opencode-antigravity-auth (PR #574)
var oauthProviderModelPresets = map[string][]oauthModelPreset{
	// ── OpenAI ────────────────────────────────────────────────────────────────
	// GPT-5.6 GA'd 2026-07-09 as three named tiers (Sol/Terra/Luna), replacing
	// the flat 5.5/5.4/5.4-mini lineup below them. Bare "gpt-5.6" aliases to
	// Sol, but we use the explicit tier IDs for clarity, matching the old
	// "-mini" suffix style.
	// gpt-5.6-sol   = flagship / frontier reasoning, Codex CLI default, $5/$30 per MTok
	// gpt-5.6-terra = balanced intelligence/cost, $2.50/$15 per MTok
	// gpt-5.6-luna  = fast, cheap, high-volume, $1/$6 per MTok
	oauthProviderOpenAI: {
		{Label: "GPT-5.6 Sol", ModelID: config.DefaultOpenAIModel},
		{Label: "GPT-5.6 Terra", ModelID: "openai/gpt-5.6-terra"},
		{Label: "GPT-5.6 Luna", ModelID: "openai/gpt-5.6-luna"},
	},

	// ── Anthropic ─────────────────────────────────────────────────────────────
	// claude-fable-5   = Mythos class, released 2026-06-09 (top tier, $10/$50/MTok)
	// claude-sonnet-5  = recommended production default, replaces Sonnet 4.6 ($3/$15/MTok,
	//                    $2/$10 intro through 2026-08-31)
	// claude-opus-4.8  = heavy reasoning ($5/$25/MTok)
	// claude-haiku-4.5 = low-latency, low-cost ($1/$5/MTok)
	oauthProviderAnthropic: {
		{Label: "Claude Fable 5", ModelID: "anthropic/claude-fable-5"},
		{Label: "Claude Sonnet 5", ModelID: config.DefaultAnthropicModel},
		{Label: "Claude Opus 4.8", ModelID: "anthropic/claude-opus-4.8"},
		{Label: "Claude Haiku 4.5", ModelID: "anthropic/claude-haiku-4.5"},
	},

	// ── Google Antigravity ────────────────────────────────────────────────────
	// Model IDs MUST match what v1internal:fetchAvailableModels returns for the
	// account — the backend 404s ("NOT_FOUND: Requested entity was not found")
	// on any ID it doesn't expose. The provider strips the "antigravity/" prefix.
	// Labels mirror the Antigravity app's model picker; the IDs are the API IDs
	// (the API's displayName differs from the ID, e.g. id gemini-3.5-flash-low →
	// "Gemini 3.5 Flash (Medium)"). All entries verified live (2026-06-28) with a
	// tool-laden request — each returns a real tool call, not just a ping.
	// Gotchas: bare "gemini-3-pro"/"gemini-3.1-pro" do NOT exist (404); the High
	// Pro tier works via "gemini-pro-agent" — the literal "gemini-3.1-pro-high" ID
	// returns INVALID_ARGUMENT via this code path. gemini-3-flash-preview is a
	// Gemini CLI quota model (different API) — NOT valid here.
	// Note: the exact ID set is rollout-dependent. If a chip 404s, re-check with
	// fetchAvailableModels for the account and update the IDs here.
	//
	// Gemini 3.6 Flash added 2026-07-22 (Google GA'd it 2026-07-21; spotted in
	// Antigravity's own tieredModelIds config as a single ID, unlike 3.5 which
	// exposes separate -low/-extra-low/-agent IDs per tier). "Flash" replaces
	// 3.5 as Antigravity's tiered "flash" family; there is only one ID here —
	// thinking effort is controlled server-side/internally, not via separate
	// High/Medium/Low model IDs, so we only list one chip for it.
	oauthProviderGoogleAntigravity: {
		{Label: "Gemini 3.6 Flash", ModelID: "antigravity/gemini-3.6-flash-tiered"},
		{Label: "Gemini 3 Flash", ModelID: config.DefaultAntigravityModel},
		{Label: "Gemini 3.5 Flash (Medium)", ModelID: "antigravity/gemini-3.5-flash-low"},
		{Label: "Gemini 3.5 Flash (High)", ModelID: "antigravity/gemini-3-flash-agent"},
		{Label: "Gemini 3.5 Flash (Low)", ModelID: "antigravity/gemini-3.5-flash-extra-low"},
		{Label: "Gemini 3.1 Pro (High)", ModelID: "antigravity/gemini-pro-agent"},
		{Label: "Gemini 3.1 Pro (Low)", ModelID: "antigravity/gemini-3.1-pro-low"},
		{Label: "Claude Sonnet 4.6 (Thinking)", ModelID: "antigravity/claude-sonnet-4-6"},
		{Label: "Claude Opus 4.6 (Thinking)", ModelID: "antigravity/claude-opus-4-6-thinking"},
		{Label: "GPT-OSS 120B (Medium)", ModelID: "antigravity/gpt-oss-120b-medium"},
	},
}

// defaultModelForProvider returns the model ID to use when a provider is first
// connected and no existing model_list entry exists for it.
// Keep in sync with the first entry of each provider block above.
func defaultModelForProvider(provider string) string {
	switch provider {
	case oauthProviderOpenAI:
		return config.DefaultOpenAIModel
	case oauthProviderAnthropic:
		return config.DefaultAnthropicModel // sonnet is recommended production default
	case oauthProviderGoogleAntigravity:
		return config.DefaultAntigravityModel
	case oauthProviderGoogleGemini:
		return config.DefaultGeminiModel
	default:
		return ""
	}
}

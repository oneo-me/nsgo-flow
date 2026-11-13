package main

import (
	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/ui"

	"github.com/oneo-me/nsgo-scenra/agent"
	"github.com/oneo-me/nsgo-scenra/locales"
)

// The storage keys of the provider's settings.
const (
	keyBaseURL = "provider.baseURL"
	keyAPIKey  = "provider.apiKey"
	keyModel   = "provider.model"
)

const defaultBaseURL = "https://api.openai.com/v1"

// provider is the provider the settings name, read from storage, which
// it follows.
func provider(cx *core.Scope) agent.Provider {
	st := cx.Storage()
	url, ok := st.GetItem(keyBaseURL)
	if !ok {
		url = defaultBaseURL
	}
	key, _ := st.GetItem(keyAPIKey)
	model, _ := st.GetItem(keyModel)
	return agent.Provider{BaseURL: url, APIKey: key, Model: model}
}

// statsSwitch turns the usage statistics on or off.
func statsSwitch(cx *core.Scope) core.Node {
	on := core.NewSignal(cx, statsOn(cx))
	return ui.Switch{Checked: on.Get, OnChange: func(v bool) { on.Set(v); setStatsOn(cx, v) },
		Label: locales.For(cx).StatsSend()}
}

// settingsDialog edits the provider, saving each field as it is typed,
// and turns the usage statistics on or off.
func settingsDialog(cx *core.Scope, open *core.Signal[bool]) core.Node {
	tr := locales.For(cx)
	colors := ui.ColorsOf(cx)
	st := cx.Storage()
	field := func(title, hint, key, fallback, placeholder string, masked bool, testID string) core.Node {
		value := func() string {
			if v, ok := st.GetItem(key); ok {
				return v
			}
			return fallback
		}
		return core.View{Style: core.Style{Gap: 6}, Children: []core.Node{
			ui.Text(title, 13, ui.Medium, colors.Text),
			ui.Input{Value: value, OnChange: func(s string) { st.SetItem(key, s) }, Placeholder: placeholder,
				Masked: masked, NoAutoCorrect: true, AutoCapitalize: core.AutoCapitalizeNone, Label: title, TestID: testID},
			core.Text{Content: hint, Style: ui.TypeCaption, StyleFunc: ui.Colored(colors.Muted)},
		}}
	}
	return ui.Dialog{Open: open.Get, OnOpenChange: func(o bool, _ base.Reason) { open.Set(o) }, TestID: "settings",
		Title: tr.SettingsTitle(), Style: core.Style{Width: core.Pt(460)},
		Children: []core.Node{
			core.View{Style: core.Style{Gap: 4}, Children: []core.Node{
				ui.Text(tr.ProviderSection(), 15, ui.Semibold, colors.Text),
				core.Text{Content: tr.ProviderDescription(), Style: ui.TypeCaption, StyleFunc: ui.Colored(colors.Muted)},
			}},
			field(tr.BaseURL(), tr.BaseURLHint(), keyBaseURL, defaultBaseURL, defaultBaseURL, false, "base-url"),
			field(tr.ApiKey(), tr.ApiKeyHint(), keyAPIKey, "", "sk-…", true, "api-key"),
			field(tr.Model(), tr.ModelHint(), keyModel, "", "gpt-4.1", false, "model"),
			core.View{Style: core.Style{Gap: 4}, Children: []core.Node{
				ui.Text(tr.StatsSection(), 15, ui.Semibold, colors.Text),
				core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 12}, Children: []core.Node{
					core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, Gap: 2}, Children: []core.Node{
						ui.Text(tr.StatsSend(), 13, ui.Medium, colors.Text),
						core.Text{Content: tr.StatsDetail(), Style: ui.TypeCaption, StyleFunc: ui.Colored(colors.Muted)},
					}},
					statsSwitch(cx),
				}},
			}},
		},
		Footer: []core.Node{ui.Button{Text: tr.Done(), TestID: "settings-done", OnPress: func() { open.Set(false) }}}}
}

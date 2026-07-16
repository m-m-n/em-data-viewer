package app

import (
	"strings"

	"data-viewer/internal/config"
	"data-viewer/internal/markup"
	"data-viewer/internal/runner"
	"data-viewer/internal/ui"
)

// reload performs one config (re)load cycle: load from path, record the
// outcome in holder (last-good retention across failed reloads), and apply
// the result to the UI/runner.
func reload(path string, holder *config.Holder, u UI, r Runner) {
	cfg, err := config.Load(path)
	holder.Update(cfg, err)
	applyConfig(cfg, err, u, r)
}

// applyConfig translates one load outcome into UI/runner side effects
// (AC-2, AC-3): on success, it clears the banner and applies the window
// settings, section view models and section list (to the runner) derived
// from cfg; on failure, it only sets the banner — window, sections and the
// runner's section list are left exactly as they were (last-good
// behavior, FR9).
func applyConfig(cfg config.Config, err error, u UI, r Runner) {
	if err != nil {
		u.SetConfigError(err.Error())
		return
	}

	u.SetConfigError("")
	u.ApplyWindow(ui.WindowSettings{
		Width:  cfg.Window.Width,
		Height: cfg.Window.Height,
		Style:  cfg.Window.Style,
	})
	u.SetSections(sectionViewModels(cfg.Sections))
	r.Apply(runnerSections(cfg.Sections))
	rehydrateSections(cfg.Sections, u, r)
}

// rehydrateSections re-populates the freshly (re)created section widgets
// from the runner's retained per-section state immediately after
// SetSections, closing the blank window a style-only reload would
// otherwise leave until the next runner event. It uses the same
// render convention as the append event path (events.go's
// translateEvent/KindAppend): join the runner's buffered lines and pass
// them through markup.Render before UpdateSection. Sections with no
// retained content (e.g. an interval section's last output, which the
// runner does not retain between runs) are left untouched; they fill in
// on the next run/event as before.
func rehydrateSections(sections []config.SectionConfig, u UI, r Runner) {
	for _, s := range sections {
		key := s.Key()
		lines := r.Snapshot(key)
		if len(lines) == 0 {
			continue
		}
		rendered, _ := markup.Render(strings.Join(lines, "\n"))
		u.UpdateSection(key, rendered, false)
	}
}

// sectionViewModels translates config sections into UI view models,
// preserving config order (display order comes from slice order; Key is
// identity, not order).
func sectionViewModels(sections []config.SectionConfig) []ui.SectionViewModel {
	out := make([]ui.SectionViewModel, len(sections))
	for i, s := range sections {
		out[i] = ui.SectionViewModel{
			Key:   s.Key(),
			Label: s.Name,
			Style: s.Style,
		}
	}
	return out
}

// runnerSections translates config sections into the runner package's
// local SectionConfig. internal/runner does not import internal/config
// (layer rule); the app layer owns this conversion.
func runnerSections(sections []config.SectionConfig) []runner.SectionConfig {
	out := make([]runner.SectionConfig, len(sections))
	for i, s := range sections {
		out[i] = runner.SectionConfig{
			Key:      s.Key(),
			Command:  s.Command,
			Mode:     runner.Mode(s.Mode),
			Interval: s.Interval,
			MaxLines: s.MaxLines,
		}
	}
	return out
}

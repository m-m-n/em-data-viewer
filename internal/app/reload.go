package app

import (
	"sync"

	"data-viewer/internal/config"
	"data-viewer/internal/runner"
	"data-viewer/internal/ui"
)

// generationTracker guards the dispatched UI-projection callback against a
// stale reload overwriting a newer one that has already been applied
// (Rework Round 1: reload pipeline, AC-5). Every triggered reload is
// assigned an increasing generation number (via next) before its
// off-main-loop work starts, so generation order reflects trigger order
// regardless of how long each reload's work takes; markApplied then lets
// only the highest generation seen so far actually reach the UI.
type generationTracker struct {
	mu      sync.Mutex
	counter uint64
	applied uint64
}

// newGenerationTracker returns a tracker with no generation issued or
// applied yet.
func newGenerationTracker() *generationTracker {
	return &generationTracker{}
}

// next returns the next generation number. Call it once per triggered
// reload, before starting its (possibly slow) off-main-loop work, so
// generation order reflects trigger order.
func (g *generationTracker) next() uint64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.counter++
	return g.counter
}

// markApplied reports whether gen may be applied to the UI: false if a
// newer generation has already been applied (gen arrived stale — an older
// reload completing after a newer one, AC-5), in which case the caller
// must skip its UI mutations entirely; true otherwise, recording gen as
// the new high-water mark.
func (g *generationTracker) markApplied(gen uint64) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if gen < g.applied {
		return false
	}
	g.applied = gen
	return true
}

// reload performs one config (re)load cycle (Rework Round 1: reload
// pipeline). Loading the file and updating holder happen on the calling
// goroutine — the watcher goroutine, or Run's startup goroutine — never
// the GTK main loop; holder becomes the single authoritative source read
// immediately after (its Config/Err, not the local load result), so it is
// never a write-only duplicate (AC-6). The rest of the work is delegated
// to applyConfig.
func reload(path string, holder *config.Holder, r Runner, store *sectionStore, gen uint64, gens *generationTracker, u UI, dispatch func(func())) {
	cfg, err := config.Load(path)
	holder.Update(cfg, err)
	applyConfig(holder.Config(), holder.Err(), r, store, gen, gens, u, dispatch)
}

// applyConfig translates one load outcome into runner/UI side effects
// (FR9, AC-4): on failure, only a banner update is dispatched — window,
// sections and the runner's section list are left exactly as they were
// (last-good behavior). On success, the runner is reconciled and the
// rehydration data is computed immediately, on the calling goroutine, off
// the GTK main loop; only the resulting UI projection (banner clear,
// window settings, keep_above, section view models, rehydration updates)
// is handed to dispatch, gated by gens.markApplied so a stale reload cannot
// overwrite a newer one (AC-5). The keep_above value is applied every
// successful reload (startup and every watch-triggered reload alike,
// window-drag-keep-above AC-1); applying the same value repeatedly is a
// deliberate no-op at the UI layer rather than something this function
// special-cases (window-drag-keep-above AC-4, Design: "must be a no-op
// visually").
func applyConfig(cfg config.Config, err error, r Runner, store *sectionStore, gen uint64, gens *generationTracker, u UI, dispatch func(func())) {
	if err != nil {
		dispatch(func() {
			if !gens.markApplied(gen) {
				return
			}
			u.SetConfigError(err.Error())
		})
		return
	}

	r.Apply(runnerSections(cfg.Sections))
	sections := sectionViewModels(cfg.Sections)
	rehydrations := rehydrationsFor(cfg.Sections, store)
	window := ui.WindowSettings{
		Width: cfg.Window.Width,
		Style: cfg.Window.Style,
	}

	dispatch(func() {
		if !gens.markApplied(gen) {
			return
		}
		u.SetConfigError("")
		u.ApplyWindow(window)
		applyKeepAbove(u, cfg.Window.KeepAbove)
		u.SetSections(sections)
		for _, rh := range rehydrations {
			u.UpdateSection(rh.key, rh.text, rh.isError)
		}
	})
}

// sectionRehydration is one section's presentation-store state to
// re-apply to a freshly (re)created widget immediately after SetSections.
type sectionRehydration struct {
	key     string
	text    string
	isError bool
}

// rehydrationsFor computes the UI updates needed to restore section
// content that SetSections' widget rebuild would otherwise leave blank
// until the next runner event: the presentation store's last rendered
// state per section key (AC-1) — this covers an interval section's last
// output and a failed section's terminal error display, neither of which
// the runner itself retains between runs (unlike a stream section's
// buffer, read via Runner.Snapshot in events.go). Sections with no
// recorded state are left out; they fill in on the next run/event as
// before.
func rehydrationsFor(sections []config.SectionConfig, store *sectionStore) []sectionRehydration {
	var out []sectionRehydration
	for _, s := range sections {
		if s.IsBreak {
			continue
		}
		key := s.Key()
		st, ok := store.get(key)
		if !ok {
			continue
		}
		out = append(out, sectionRehydration{key: key, text: st.text, isError: st.isError})
	}
	return out
}

// sectionViewModels translates config sections into UI view models,
// preserving config order (display order comes from slice order; Key is
// identity, not order) — including break entries (section-row-break
// task0001 FR2, FR4): one view model per config entry so the ui layer's
// row-break positions line up with the original config positions;
// a break entry becomes a bare {IsBreak: true} marker.
func sectionViewModels(sections []config.SectionConfig) []ui.SectionViewModel {
	out := make([]ui.SectionViewModel, len(sections))
	for i, s := range sections {
		if s.IsBreak {
			out[i] = ui.SectionViewModel{IsBreak: true}
			continue
		}
		out[i] = ui.SectionViewModel{
			Key:   s.Key(),
			Label: s.Name,
			Style: s.Style,
		}
	}
	return out
}

// runnerSections translates config sections into the runner package's
// local SectionConfig, excluding break entries (section-row-break task0001
// FR2, AC-4): a break never starts a process. internal/runner does not
// import internal/config (layer rule); the app layer owns this conversion.
func runnerSections(sections []config.SectionConfig) []runner.SectionConfig {
	out := make([]runner.SectionConfig, 0, len(sections))
	for _, s := range sections {
		if s.IsBreak {
			continue
		}
		out = append(out, runner.SectionConfig{
			Key:      s.Key(),
			Command:  s.Command,
			Mode:     runner.Mode(s.Mode),
			Interval: s.Interval,
			MaxLines: s.MaxLines,
		})
	}
	return out
}

import GLib from 'gi://GLib';
import {Extension} from 'resource:///org/gnome/shell/extensions/extension.js';

// Shared Components (IMPLEMENTATION.md): must equal the Makefile's APP_ID.
const TARGET_APP_ID = 'io.github.m_m_n.EmDataViewer';
// Process-identity verification (FR2 addition): the executable name an
// id-matched window's PID must resolve to via /proc before it is trusted.
const TARGET_EXECUTABLE_NAME = 'em-data-viewer';

// Either-id matching (FR2 revision): a window matches as soon as either
// identifier equals the target id. A non-match is final only once both
// identifiers are set and neither matches; while one identifier is still
// unset, the decision is 'pending' and the caller must keep watching.
function decideMatch(window) {
    const gtkAppId = window.get_gtk_application_id();
    const wmClass = window.get_wm_class();
    if (gtkAppId === TARGET_APP_ID || wmClass === TARGET_APP_ID)
        return 'match';
    if (gtkAppId && wmClass)
        return 'no-match';
    return 'pending';
}

// Process-identity verification (FR2 addition): the single decision point,
// shared by the enable-time scan, window-created path, and re-check path,
// that confirms the compositor-reported PID resolves - via the proc
// filesystem - to the em-data-viewer executable before a window is trusted.
// No PID, an unreadable link, or a different executable name all verify as
// false; there is no retry loop for this check.
//
// Deleted-marker tolerance (FR2 detail, review round 2): the kernel appends
// ' (deleted)' to the /proc/PID/exe link target when the on-disk binary was
// replaced or removed after the process started (e.g. a package upgrade
// while em-data-viewer keeps running). Strip it before the basename
// comparison so a still-running, already-updated instance keeps verifying.
const DELETED_MARKER = ' (deleted)';

function isVerifiedProcess(window) {
    const pid = window.get_pid();
    if (!pid || pid <= 0)
        return false;

    let exePath;
    try {
        exePath = GLib.file_read_link(`/proc/${pid}/exe`);
    } catch (e) {
        return false;
    }

    if (!exePath)
        return false;

    const resolvedPath = exePath.endsWith(DELETED_MARKER)
        ? exePath.slice(0, -DELETED_MARKER.length)
        : exePath;

    return GLib.path_get_basename(resolvedPath) === TARGET_EXECUTABLE_NAME;
}

export default class KeepAboveExtension extends Extension {
    enable() {
        // window -> handler ids for its pending id watch (FR2 late-id handling).
        this._windowWatches = new Map();
        // window -> its 'unmanaged' handler id, for exactly the windows this
        // extension itself set above (FR4 revision, round 2): disable()
        // reverts this tracked set and nothing else - no re-deriving
        // "matching windows" with a separate matcher.
        this._appliedWindows = new Map();

        this._displayHandlerId = global.display.connect('window-created',
            (_display, window) => this._maybeApply(window));

        for (const window of global.display.list_all_windows())
            this._maybeApply(window);
    }

    disable() {
        global.display.disconnect(this._displayHandlerId);
        this._displayHandlerId = null;

        for (const window of this._windowWatches.keys())
            this._detachWatch(window);
        this._windowWatches = null;

        for (const [window, unmanagedId] of this._appliedWindows) {
            window.disconnect(unmanagedId);
            window.unmake_above();
        }
        this._appliedWindows = null;
    }

    _maybeApply(window) {
        const decision = decideMatch(window);
        if (decision === 'pending') {
            // App id not fully known yet (Wayland): watch until both
            // identifiers are available, then decide once (FR2 late-id
            // handling, either-id matching).
            const ids = [
                window.connect('notify::gtk-application-id', () => this._recheck(window)),
                window.connect('notify::wm-class', () => this._recheck(window)),
                window.connect('unmanaged', () => this._detachWatch(window)),
            ];
            this._windowWatches.set(window, ids);
            return;
        }

        this._applyIfMatched(window, decision);
    }

    _recheck(window) {
        const decision = decideMatch(window);
        if (decision === 'pending')
            return;

        this._detachWatch(window);
        this._applyIfMatched(window, decision);
    }

    _applyIfMatched(window, decision) {
        if (decision !== 'match' || !isVerifiedProcess(window) || window.above)
            return;

        window.make_above();
        // Record the window this call actually set above (FR4 revision) so
        // disable() reverts exactly this set. A window already above (user
        // set it manually pre-enable) returned above and is never recorded.
        const unmanagedId = window.connect('unmanaged', () => {
            window.disconnect(unmanagedId);
            this._appliedWindows.delete(window);
        });
        this._appliedWindows.set(window, unmanagedId);
    }

    _detachWatch(window) {
        const ids = this._windowWatches.get(window);
        if (!ids)
            return;

        for (const id of ids)
            window.disconnect(id);
        this._windowWatches.delete(window);
    }
}

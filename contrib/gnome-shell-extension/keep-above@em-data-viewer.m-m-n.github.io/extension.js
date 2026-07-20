import {Extension} from 'resource:///org/gnome/shell/extensions/extension.js';

// Shared Components (IMPLEMENTATION.md): must equal the Makefile's APP_ID.
const TARGET_APP_ID = 'io.github.m_m_n.EmDataViewer';

function appIdOf(window) {
    return window.get_gtk_application_id() || window.get_wm_class();
}

export default class KeepAboveExtension extends Extension {
    enable() {
        // window -> handler ids for its pending id watch (FR2 late-id handling).
        this._windowWatches = new Map();

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

        for (const window of global.display.list_all_windows()) {
            if (appIdOf(window) === TARGET_APP_ID && window.above)
                window.unmake_above();
        }
    }

    _maybeApply(window) {
        const appId = appIdOf(window);
        if (appId) {
            if (appId === TARGET_APP_ID && !window.above)
                window.make_above();
            return;
        }

        // App id not set yet (Wayland): watch until it arrives, then decide once.
        const ids = [
            window.connect('notify::gtk-application-id', () => this._recheck(window)),
            window.connect('notify::wm-class', () => this._recheck(window)),
            window.connect('unmanaged', () => this._detachWatch(window)),
        ];
        this._windowWatches.set(window, ids);
    }

    _recheck(window) {
        const appId = appIdOf(window);
        if (!appId)
            return;

        this._detachWatch(window);
        if (appId === TARGET_APP_ID && !window.above)
            window.make_above();
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

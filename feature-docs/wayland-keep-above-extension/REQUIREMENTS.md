---
title: "Wayland keep-above GNOME Shell 拡張"
created_date: 2026-07-21
status: draft
---

# Wayland keep-above GNOME Shell 拡張 - 要件定義書

## 1. 概要

### 1.1 背景

- アプリの `window.keep_above` 設定は X11/XWayland のみ有効で、ネイティブ Wayland では仕様として inert(feature-docs/window-drag-keep-above/SPEC.md FR4)
- Wayland にはクライアントが always-on-top を要求する標準プロトコルが存在しない
- GNOME/Mutter ではコンポジタ内部 API `Meta.Window.make_above()` を Shell 拡張から呼ぶ方法がある(tmp/wayland-keep-above-research-2026-07-21.md)

### 1.2 目的

ネイティブ Wayland セッションで、em-data-viewer のウィンドウをログイン直後の起動時から自動的に最前面(always-on-top)にする。

### 1.3 スコープ

- GNOME Shell 拡張(自作ミニ拡張)をリポジトリに同梱する
- Makefile にインストール/アンインストールターゲットを追加する
- 対象外: .deb への拡張同梱、GNOME 以外のデスクトップ環境、アプリ本体(Go)の変更

## 2. ユースケース

### UC01: ログイン時の自動最前面化

**アクター**: em-data-viewer をログイン時から常時表示したいユーザー

**事前条件**:
- GNOME Shell 50 の Wayland セッション
- 拡張がインストール・有効化済み

**基本フロー**:
1. ユーザーがログインする
2. em-data-viewer が起動しウィンドウが作られる
3. 拡張が app_id `io.github.m_m_n.EmDataViewer` のウィンドウを検出する
4. 拡張が `make_above()` を呼び、ウィンドウが最前面に維持される

**代替フロー**:
- 拡張を後から有効化した場合: すでに開いている該当ウィンドウにも適用される

**事後条件**:
- em-data-viewer のウィンドウが他のウィンドウより前面に維持される

### UC02: 拡張のインストール/アンインストール

**アクター**: 開発者・ユーザー

**基本フロー(インストール)**:
1. `make install-extension` を実行する
2. 拡張が `~/.local/share/gnome-shell/extensions/` にコピーされる
3. `gnome-extensions enable` が実行される
4. 新規インストール時は再ログインが必要な旨のメッセージが表示される

**基本フロー(アンインストール)**:
1. `make uninstall-extension` を実行する
2. 拡張が disable され、拡張ディレクトリが削除される

## 3. 機能要件

### 3.1 機能一覧

| ID | 機能名 | 説明 | 優先度 |
|----|--------|------|--------|
| F01 | 拡張本体 | app_id 一致ウィンドウを `make_above()` する GNOME Shell 拡張 | 高 |
| F02 | 既存ウィンドウへの適用 | enable 時にすでに開いているウィンドウにも適用する | 高 |
| F03 | disable 時の後始末 | シグナル切断と、拡張が above にしたウィンドウの解除 | 高 |
| F04 | install-extension | インストールと enable を行う Makefile ターゲット | 高 |
| F05 | uninstall-extension | disable と拡張ディレクトリ削除を行う Makefile ターゲット | 中 |

### 3.2 機能詳細

#### F01: 拡張本体

- UUID: `keep-above@em-data-viewer.m-m-n.github.io`
- 配置: `contrib/gnome-shell-extension/keep-above@em-data-viewer.m-m-n.github.io/`
- `window-created` シグナルで app_id `io.github.m_m_n.EmDataViewer` のウィンドウを検出し `make_above()` を呼ぶ
- `make_above()` の適用前にウィンドウのプロセス実体(PID → 実行ファイル名 `em-data-viewer`)を検証する
- 適用条件はアプリの config `window.keep_above` とは連動しない
- metadata.json の shell-version は `50` のみ

#### F02: 既存ウィンドウへの適用

- enable 時に全ウィンドウを走査し、該当 app_id のウィンドウに `make_above()` する

#### F03: disable 時の後始末

- 接続したシグナルを全て切断する
- 拡張が above にした該当ウィンドウは `unmake_above()` で解除する

#### F04: install-extension

- 拡張ソースを `~/.local/share/gnome-shell/extensions/{UUID}/` にコピーする
- `gnome-extensions enable {UUID}` を実行する
- 新規インストールで Shell が拡張を未認識の場合、再ログインが必要な旨を案内する

#### F05: uninstall-extension

- `gnome-extensions disable {UUID}` を実行する(失敗しても継続)
- `~/.local/share/gnome-shell/extensions/{UUID}/` を削除する

## 4. 非機能要件

### 4.1 互換性要件

- GNOME Shell 50 / Wayland セッションで動作する
- GNOME Shell 45+ の ESM 形式で記述する

### 4.2 保守性要件

- アプリ本体(Go)のコード・挙動は変更しない
- 拡張は自己完結(外部依存なし)とする
- 既存の `go test ./...` に影響を与えない

## 5. 制約条件

- Wayland には always-on-top の標準プロトコルがないため、GNOME Shell 拡張(Mutter 内部 API)方式に限定される
- 拡張の有効化はユーザーごとに必要(システム全体配布はしない)
- GJS 拡張は自動テストの対象外(GUI 挙動は手動確認する既存方針に従う)

## 6. 成功基準

- [ ] Wayland セッションでログイン直後から em-data-viewer が最前面に維持される
- [ ] `make install-extension` / `make uninstall-extension` が動作する
- [ ] 拡張 disable で above 状態が解除される
- [ ] `go test ./...` が引き続き pass する

## 7. 確認事項

### 7.1 確認済み事項

- [x] 適用条件: 無条件で常に適用(アプリ config `window.keep_above` とは連動しない)
- [x] 配置ディレクトリ: `contrib/gnome-shell-extension/`
- [x] `make install-extension` は enable まで自動で行う
- [x] shell-version: `50` のみ
- [x] enable 時に既存ウィンドウにも適用する
- [x] `make uninstall-extension` も作る
- [x] UUID: `keep-above@em-data-viewer.m-m-n.github.io`
- [x] レビュー round 1 対応: F1(either-id マッチング)・F2(既知インストール時の enable 失敗伝播)・F3(PID による実行ファイル検証)を修正する

### 7.2 未確認・保留事項

なし

## 8. 参考資料

- 調査レポート: tmp/wayland-keep-above-research-2026-07-21.md
- Mutter API: https://mutter.gnome.org/meta/method.Window.make_above.html
- 既存仕様: feature-docs/window-drag-keep-above/SPEC.md (FR4)

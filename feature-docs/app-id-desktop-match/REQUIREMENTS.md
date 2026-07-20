---
title: "application ID と .desktop ファイル名の一致"
created_date: 2026-07-20
status: draft
---

# application ID と .desktop ファイル名の一致 - 要件定義書

## 1. 概要

### 1.1 背景

Wayland セッション（GNOME Shell）では、dock・Alt-Tab のアイコン解決がウィンドウの app_id と .desktop ファイル名のマッチングで行われる。GTK4 + GtkApplication のアプリでは app_id は application ID（現在 `org.dataviewer.Widget`）になるが、.desktop は `em-data-viewer.desktop` のため一致せず、デフォルトの歯車アイコンが表示される。`StartupWMClass` は X11 の WM_CLASS 用で Wayland では機能しない。

### 1.2 目的

application ID をアプリ固有の `io.github.m_m_n.EmDataViewer` に変更し、.desktop ファイル名をこれに一致させることで、Wayland の dock にアプリアイコンを表示する。

### 1.3 スコープ

- application ID の変更（internal/ui）
- .desktop ファイル名の変更（Makefile の dpkg ターゲット）
- アイコン名 `em-data-viewer`（Icon= / hicolor / gresource / SetIconName）は変更しない

## 2. ユースケース

### 2.1 ユースケース詳細

#### UC01: Wayland の dock にアイコンが表示される

**アクター**: アプリ利用者

**事前条件**:
- deb パッケージがインストールされている
- Wayland セッション（GNOME Shell）

**基本フロー**:
1. アプリを起動する
2. app_id `io.github.m_m_n.EmDataViewer` と `io.github.m_m_n.EmDataViewer.desktop` がファイル名マッチする
3. .desktop の `Icon=em-data-viewer` から dock にアイコンが表示される

## 3. 機能要件

### 3.1 機能一覧

| ID | 機能名 | 説明 | 優先度 |
|----|--------|------|--------|
| F01 | application ID 変更 | `org.dataviewer.Widget` → `io.github.m_m_n.EmDataViewer` | 高 |
| F02 | .desktop ファイル名変更 | `em-data-viewer.desktop` → `io.github.m_m_n.EmDataViewer.desktop`（内容は不変） | 高 |

### 3.2 機能詳細

#### F01: application ID 変更

**説明**: `internal/ui` の applicationID 定数を `io.github.m_m_n.EmDataViewer` に変更する。GApplication ID として妥当な形式（4 要素、各要素が有効文字）であることを確認済み。

#### F02: .desktop ファイル名変更

**説明**: Makefile の dpkg ターゲットが生成する .desktop のファイル名のみ変更する。内容（Name / Comment / Exec / Icon=em-data-viewer / Terminal / Type / Categories / StartupWMClass=em-data-viewer）は変更しない。StartupWMClass は X11 セッションでのマッチング用に維持する。

## 4. 非機能要件

### 4.1 互換性要件

- アイコン名 `em-data-viewer`（hicolor ファイル名・gresource エイリアス・SetIconName・Icon=）は一切変更しない
- deb 更新時、旧 `em-data-viewer.desktop` は dpkg のファイルリスト置換で自動削除される

## 5. 成功基準

### 5.1 受け入れ基準

- [ ] applicationID 定数が `io.github.m_m_n.EmDataViewer` になっている
- [ ] deb に `usr/share/applications/io.github.m_m_n.EmDataViewer.desktop` が含まれ、`em-data-viewer.desktop` は含まれない
- [ ] .desktop の内容が従来と同一（ファイル名のみ変更）
- [ ] Wayland セッションで dock にアプリアイコンが表示される（手動確認）

## 6. 確認事項

### 6.1 確認済み事項

- [x] 原因: Wayland では app_id と .desktop ファイル名のマッチングが必須（StartupWMClass は X11 専用）。emterm が表示できるのは GTK3 + GtkApplication 不使用で app_id がバイナリ名になるため
- [x] application ID: `io.github.m_m_n.EmDataViewer`（net.laser5 はこのアプリでは使わない）
- [x] アイコン名は em-data-viewer のまま維持

### 6.2 未確認・保留事項

なし

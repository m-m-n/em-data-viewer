---
title: "アプリアイコン"
created_date: 2026-07-19
status: draft
---

# アプリアイコン - 要件定義書

## 1. 概要

### 1.1 背景

em-data-viewer にはアプリアイコンが存在せず、deb パッケージにも .desktop ファイルが含まれていない。そのためアプリ一覧・タスクバーにアイコンが表示されない。

### 1.2 目的

アプリアイコンを SVG で作成し、ビルド時に PNG へ変換して deb パッケージに同梱する。アプリ一覧・タスクバー・ウィンドウにアイコンが表示される状態にする。

### 1.3 スコープ

- SVG アイコンソースの作成と git 管理
- ビルド時の SVG → PNG 変換（生成物は git 管理外）
- deb パッケージへの hicolor アイコン・.desktop ファイル同梱
- 実行時のウィンドウアイコン設定

## 2. ビジネス要件

### 2.1 対象ユーザー

| ユーザータイプ | 説明 |
|----------------|------|
| アプリ利用者 | deb パッケージをインストールして em-data-viewer を使う人 |

### 2.2 期待される効果

- アプリ一覧・タスクバーでアプリを視覚的に識別できる

## 3. ユースケース

### 3.1 ユースケース一覧

| ID | ユースケース名 | アクター | 優先度 |
|----|----------------|----------|--------|
| UC01 | deb インストール後にアイコンが表示される | アプリ利用者 | 高 |
| UC02 | ビルド時にアイコン PNG が生成される | 開発者 | 高 |

### 3.2 ユースケース詳細

#### UC01: deb インストール後にアイコンが表示される

**アクター**: アプリ利用者

**事前条件**:
- deb パッケージがインストールされている

**基本フロー**:
1. デスクトップ環境のアプリ一覧に「eM Data Viewer」がアイコン付きで表示される
2. アプリを起動するとタスクバー・ウィンドウにアイコンが表示される

**事後条件**:
- hicolor テーマにアイコンが登録され、desktop database が更新されている

#### UC02: ビルド時にアイコン PNG が生成される

**アクター**: 開発者

**事前条件**:
- rsvg-convert がインストールされている

**基本フロー**:
1. ビルドを実行する
2. assets/icon.svg から各サイズの PNG が build/ 配下に生成される
3. dpkg ターゲットで PNG・SVG・.desktop が deb に同梱される

**代替フロー**:
- rsvg-convert が無い場合、ビルドはエラーで失敗する（アイコン欠落の deb を作らない）

## 4. 機能要件

### 4.1 機能一覧

| ID | 機能名 | 説明 | 優先度 |
|----|--------|------|--------|
| F01 | SVG アイコンソース | assets/icon.svg を git 管理する | 高 |
| F02 | ビルド時 PNG 変換 | rsvg-convert で 48/64/128/256 の PNG を build/ 配下に生成する | 高 |
| F03 | hicolor アイコン同梱 | deb に PNG（サイズ別）と SVG（scalable）を配置する | 高 |
| F04 | .desktop ファイル同梱 | em-data-viewer.desktop を deb に配置する | 高 |
| F05 | ウィンドウアイコン設定 | 実行時に icon-name を設定する | 中 |
| F06 | アイコンキャッシュ更新 | postinst / postrm で gtk-update-icon-cache と update-desktop-database を実行する | 中 |

### 4.2 機能詳細

#### F01: SVG アイコンソース

**説明**: アイコンの原本を `assets/icon.svg` として作成し git 管理する。ビジュアルデザインは design フェーズで決定する。

#### F02: ビルド時 PNG 変換

**説明**: rsvg-convert で `assets/icon.svg` から 48/64/128/256 px の PNG を `build/` 配下に生成する。

**エラーケース**:

| エラー | 条件 | 対応 |
|--------|------|------|
| 変換ツール不在 | rsvg-convert が見つからない | ビルドをエラーで失敗させる |

#### F03: hicolor アイコン同梱

**説明**: deb パッケージに以下を配置する。

- `/usr/share/icons/hicolor/{48x48,64x64,128x128,256x256}/apps/em-data-viewer.png`
- `/usr/share/icons/hicolor/scalable/apps/em-data-viewer.svg`

#### F04: .desktop ファイル同梱

**説明**: `/usr/share/applications/em-data-viewer.desktop` を配置する。

- `Name=eM Data Viewer`
- `Exec=em-data-viewer`
- `Icon=em-data-viewer`
- `StartupWMClass=em-data-viewer`
- `Terminal=false`
- `Type=Application`

application ID（`org.dataviewer.Widget`）は変更せず、単純名 `em-data-viewer` でアイコン・.desktop を揃え、StartupWMClass でタスクバーとマッチングさせる。

#### F05: ウィンドウアイコン設定

**説明**: 実行時にウィンドウの icon-name として `em-data-viewer` を設定し、インストール済み hicolor テーマからアイコンが解決されるようにする。

#### F06: アイコンキャッシュ更新

**説明**: deb の postinst / postrm で `gtk-update-icon-cache` と `update-desktop-database` を実行する（コマンド不在時は無視して成功させる）。

## 5. 非機能要件

### 5.1 保守性要件

- PNG 等の生成物は `build/` 配下に出力し git 管理しない（`build/` は gitignore 済み）
- アイコンの原本は `assets/icon.svg` の 1 ファイルのみとする

## 6. 制約条件

### 6.1 技術的制約

- SVG → PNG 変換は rsvg-convert を使用する
- ビルドは Makefile（build / dpkg ターゲット）で行う

## 7. 成功基準

### 7.1 受け入れ基準

- [ ] `assets/icon.svg` が git 管理されている
- [ ] ビルドで 48/64/128/256 の PNG が `build/` 配下に生成される
- [ ] rsvg-convert 不在時にビルドが失敗する
- [ ] deb に hicolor アイコン（PNG 4 サイズ + scalable SVG）が含まれる
- [ ] deb に em-data-viewer.desktop が含まれる
- [ ] deb に postinst / postrm が含まれ、キャッシュ更新を行う
- [ ] 実行時にウィンドウの icon-name が設定される
- [ ] git 管理下に新規バイナリファイルが増えない

## 8. テストシナリオ

### 8.1 テスト観点

- [ ] 正常系: make dpkg で deb にアイコン・.desktop・postinst / postrm が含まれる
- [ ] 異常系: rsvg-convert 不在を模した環境でビルドが失敗する
- [ ] 確認: git status で生成 PNG が untracked に現れない

## 9. 確認事項

### 9.1 確認済み事項

- [x] 適用範囲: フル適用（deb 同梱 + .desktop + 実行時ウィンドウアイコン設定）
- [x] デザイン: お任せ（design フェーズで決定、実機確認後に /em-workflow:design で調整）
- [x] 変換ツール: rsvg-convert。不在時はビルド失敗
- [x] SVG 配置: assets/icon.svg
- [x] サイズ展開: 標準セット（48/64/128/256 PNG + scalable SVG）
- [x] 命名方式: application ID は変更せず、単純名 em-data-viewer で .desktop・アイコン名を揃え StartupWMClass でマッチング（emterm 方式）
- [x] 表示名: eM Data Viewer

### 9.2 未確認・保留事項

なし

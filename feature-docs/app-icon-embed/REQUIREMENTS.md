---
title: "アプリアイコンのバイナリ組み込み"
created_date: 2026-07-19
status: draft
---

# アプリアイコンのバイナリ組み込み - 要件定義書

## 1. 概要

### 1.1 背景

app-icon feature でアイコンテーマ経由のウィンドウアイコン設定（`SetIconName`）を実装したが、deb 未インストール環境（開発中のバイナリ直接起動など）では hicolor テーマにアイコンが存在せず、dock にデフォルトの歯車アイコンが表示される。

### 1.2 目的

アイコンを GResource としてバイナリに組み込み、アイコンテーマのインストール状態に関係なく dock・タスクバー・ウィンドウにアプリアイコンが表示されるようにする。

### 1.3 スコープ

- ビルド時の GResource バンドル生成（icon.gresource）
- バイナリへの埋め込み（go:embed）と実行時のアイコンテーマ登録
- ビルド・テスト標準コマンドの Makefile 経由化
- 既存の deb 向け hicolor インストール・.desktop は変更しない

## 2. ユースケース

### 2.1 ユースケース一覧

| ID | ユースケース名 | アクター | 優先度 |
|----|----------------|----------|--------|
| UC01 | deb 未インストールでもアイコンが表示される | 開発者・利用者 | 高 |
| UC02 | make でバイナリを再現ビルドできる | 開発者 | 高 |

### 2.2 ユースケース詳細

#### UC01: deb 未インストールでもアイコンが表示される

**アクター**: 開発者・利用者

**事前条件**:
- `make build` で生成したバイナリがある（deb インストール不要）

**基本フロー**:
1. バイナリを直接起動する
2. dock・タスクバー・ウィンドウにアプリアイコンが表示される

**事後条件**:
- アイコン解決は埋め込みリソースから行われる

#### UC02: make でバイナリを再現ビルドできる

**アクター**: 開発者

**事前条件**:
- rsvg-convert と glib-compile-resources がインストールされている

**基本フロー**:
1. `make build` または `make test` を実行する
2. icon.gresource が生成され、go build / go test が成功する

**代替フロー**:
- rsvg-convert または glib-compile-resources が無い場合、ビルドはエラーで失敗する

## 3. 機能要件

### 3.1 機能一覧

| ID | 機能名 | 説明 | 優先度 |
|----|--------|------|--------|
| F01 | GResource バンドル生成 | SVG から各サイズ PNG を生成し glib-compile-resources で internal/ui/icon.gresource を生成する | 高 |
| F02 | ビルド入口の Makefile 経由化 | make build / make test（新設）が gresource 生成に依存する | 高 |
| F03 | 実行時リソース登録 | 埋め込んだ gresource を登録しアイコンテーマの検索パスに追加する | 高 |
| F04 | 生成物の git 除外 | internal/ui/icon.gresource を .gitignore に追加する | 高 |

### 3.2 機能詳細

#### F01: GResource バンドル生成

**説明**: `assets/icon.svg` から 16/32/48/64/128/256 px の PNG を `build/` 配下に生成し、gresource XML（git 管理するテキスト）に従って `internal/ui/icon.gresource` を生成する。リソース内は hicolor テーマ構造（`{S}x{S}/apps/em-data-viewer.png`）で配置する。

**エラーケース**:

| エラー | 条件 | 対応 |
|--------|------|------|
| 変換ツール不在 | rsvg-convert が見つからない | ビルドをエラーで失敗させる |
| コンパイラ不在 | glib-compile-resources が見つからない | ビルドをエラーで失敗させる |

#### F02: ビルド入口の Makefile 経由化

**説明**: `make build`（既存を拡張）と `make test`（新設）を標準入口とし、両方とも gresource 生成ターゲットに依存させる。go:embed は対象ファイル不在時にコンパイルエラーになるため、生成順序を Makefile が保証する。一度生成すればファイルが残るので、以降は素の go コマンドも通る。

#### F03: 実行時リソース登録

**説明**: `internal/ui` で icon.gresource を go:embed し、GTK 初期化後・ウィンドウ生成前にリソースを登録してアイコンテーマの検索パスに追加する。既存の `SetIconName("em-data-viewer")` はそのまま活き、テーマ未インストールでも埋め込みリソースから解決される。

#### F04: 生成物の git 除外

**説明**: `internal/ui/icon.gresource` を `.gitignore` に追加し、バイナリを git 管理しない。

## 4. 非機能要件

### 4.1 保守性要件

- バイナリファイル（PNG・gresource）は git にコミットしない
- 既存の deb 向け hicolor インストール・.desktop・postinst/postrm は変更しない（両立）

## 5. 制約条件

### 5.1 技術的制約

- GResource 生成は glib-compile-resources、PNG 生成は rsvg-convert を使用する
- go:embed の制約により icon.gresource は internal/ui/ 配下（ソースツリー内）に生成する

## 6. 成功基準

### 6.1 受け入れ基準

- [ ] `make build` で icon.gresource が生成されバイナリが完成する
- [ ] `make test` で全テストが通る
- [ ] rsvg-convert または glib-compile-resources 不在時にビルドが失敗する
- [ ] deb 未インストール環境でバイナリ直接起動時に dock・ウィンドウへアイコンが表示される
- [ ] git 管理下に新規バイナリファイルが増えない（icon.gresource は ignore される）
- [ ] 既存の deb パッケージング（hicolor / .desktop / postinst / postrm）が変わらない

## 7. テストシナリオ

### 7.1 テスト観点

- [ ] 正常系: make build → make test が成功し、icon.gresource が生成される
- [ ] 異常系: glib-compile-resources 不在を模した環境でビルドが失敗する
- [ ] 確認: git status に icon.gresource が現れない

## 8. 確認事項

### 8.1 確認済み事項

- [x] 方式: GResource + アイコンテーマ検索パス登録（X11 の _NET_WM_ICON 直接設定ではなく GTK 標準機構）
- [x] 生成物配置: internal/ui/icon.gresource（go:embed の制約によりソースツリー内）+ .gitignore 追加
- [x] サイズセット: 16/32/48/64/128/256
- [x] ツール不在時: ビルド失敗（rsvg-convert と同方針）
- [x] ビルド入口: make build / make test（新設）に統一。workflow の build_command / test_command も変更

### 8.2 未確認・保留事項

なし

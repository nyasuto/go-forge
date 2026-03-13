# PRD: GoForge — Go UNIXツールチェーン量産プロジェクト

## 概要

GoForgeは、UNIX標準コマンドおよび日常的に使うCLIツールをGoでゼロから再実装するプロジェクトである。学習・理解を目的とした「再発明」であり、既存ツールの完全互換を目指すのではなく、コア機能の再実装＋独自の改良を加える方針とする。

各ツールは独立したGoモジュール（シングルバイナリ）として実装し、Ralph Loopによる自律的な量産開発に最適化した設計とする。

## 設計原則

- **シングルバイナリ**: 各ツールは `go build` で単一バイナリを生成
- **ゼロ外部依存**: 標準ライブラリのみ（例外は明示的に記載）
- **テスト駆動**: 各機能にテーブルドリブンテストを必須とする
- **Progressive Enhancement**: Tier 1（コア機能）→ Tier 2（拡張）→ Tier 3（独自改良）の段階的実装
- **UNIX哲学準拠**: stdin/stdout パイプ対応、終了コード準拠

## リポジトリ構成

```
goforge/
├── CLAUDE.md
├── Makefile
├── go.work
├── cmd/
│   ├── gf-cat/
│   ├── gf-head/
│   ├── gf-tail/
│   ├── gf-wc/
│   ├── gf-tee/
│   ├── gf-grep/
│   ├── gf-find/
│   ├── gf-sort/
│   ├── gf-uniq/
│   ├── gf-cut/
│   ├── gf-sed/
│   ├── gf-xargs/
│   ├── gf-diff/
│   ├── gf-tree/
│   ├── gf-jq/
│   └── gf-hexdump/
└── docs/
    └── test-results/   # 各ツールのテスト結果レポート
```

## 共通仕様

### CLI引数パース
- `flag` 標準パッケージのみ使用
- `--help` で usage 表示
- `--version` でバージョン表示（バージョンは `0.1.0` 固定）
- 未知のフラグはエラー終了（exit code 2）

### I/O規約
- 引数なし or `-` → stdin から読み取り
- 引数あり → ファイルパスとして処理
- 出力は常に stdout（エラーは stderr）
- パイプ対応必須

### エラーハンドリング
- ファイルが存在しない → stderr にメッセージ、exit code 1
- パーミッションエラー → stderr にメッセージ、exit code 1
- パイプ破断（SIGPIPE）→ 静かに終了

### テスト要件
- 各ツールに `main_test.go` を必須とする
- テーブルドリブンテスト形式
- 最低テストケース: 正常系3、異常系2、エッジケース2（空入力、巨大入力、マルチバイト）
- テスト完了後、結果を `docs/test-results/gf-xxx.md` に記録する

---

## タスクリスト

> **ルール**: 上から順に1つずつ実行する。完了したら `[x]` に変更。

### Task 0: プロジェクト初期化

- [x] go.work、Makefile、docs/test-results/ ディレクトリを作成。Makefileには `build`（全ツールビルド）、`test`（全ツールテスト）、`quality`（test + vet）ターゲットを定義。

---

### Wave 1: 基礎テキスト処理

#### gf-cat（ファイル結合・表示）

- [x] Tier 1: `cmd/gf-cat/` 作成（go.mod初期化、go.workに追加）。ファイル連結表示、stdin対応、複数ファイル引数対応。テスト作成・通過。テスト結果を `docs/test-results/gf-cat.md` に記録。
- [x] Tier 2: `-n` 行番号表示、`-s` 連続空行の圧縮。テスト追加・通過。テスト結果を更新。
- [x] Tier 3: ファイル拡張子に基づくシンタックスハイライト自動検出（.go, .py, .js, .json, .yaml 対応）。テスト追加・通過。テスト結果を更新。

#### gf-head（先頭表示）

- [x] Tier 1: `cmd/gf-head/` 作成（go.mod初期化、go.workに追加）。デフォルト10行表示、`-n` 行数指定、stdin対応。テスト作成・通過。テスト結果を `docs/test-results/gf-head.md` に記録。
- [x] Tier 2: `-c` バイト数指定、複数ファイル対応（ヘッダ付き）。テスト追加・通過。テスト結果を更新。
- [x] Tier 3: ストリーミングモード — stdinから指定行数を受け取るたびにクリア＆再表示。テスト追加・通過。テスト結果を更新。

#### gf-tail（末尾表示）

- [x] Tier 1: `cmd/gf-tail/` 作成（go.mod初期化、go.workに追加）。デフォルト10行表示、`-n` 行数指定、stdin対応。テスト作成・通過。テスト結果を `docs/test-results/gf-tail.md` に記録。
- [x] Tier 2: `-f` フォローモード（ファイル追記の監視）。テスト追加・通過。テスト結果を更新。
- [x] Tier 3: `-f` + `-p パターン` でマッチ行をハイライト表示。テスト追加・通過。テスト結果を更新。

#### gf-wc（カウント）

- [x] Tier 1: `cmd/gf-wc/` 作成（go.mod初期化、go.workに追加）。行数・単語数・バイト数カウント、stdin対応。テスト作成・通過。テスト結果を `docs/test-results/gf-wc.md` に記録。
- [x] Tier 2: `-m` 文字数（rune対応）、複数ファイル合計行の表示。テスト追加・通過。テスト結果を更新。
- [x] Tier 3: `--json` JSON形式出力。テスト追加・通過。テスト結果を更新。

#### gf-tee（分岐出力）

- [x] Tier 1: `cmd/gf-tee/` 作成（go.mod初期化、go.workに追加）。stdinを読み取りstdout + 指定ファイルに書き出し。テスト作成・通過。テスト結果を `docs/test-results/gf-tee.md` に記録。
- [x] Tier 2: `-a` appendモード。テスト追加・通過。テスト結果を更新。
- [x] Tier 3: 複数ファイル同時書き出し、`--ts` タイムスタンプ付与オプション。テスト追加・通過。テスト結果を更新。

---

### Wave 2: 検索・フィルタリング

#### gf-grep（パターン検索）

- [x] Tier 1: `cmd/gf-grep/` 作成（go.mod初期化、go.workに追加）。固定文字列マッチ、マッチ行出力、stdin対応。テスト作成・通過。テスト結果を `docs/test-results/gf-grep.md` に記録。
- [x] Tier 2: 正規表現対応、`-i`（大文字小文字無視）、`-v`（反転）、`-c`（カウント）、`-n`（行番号）、`-r`（再帰検索）。テスト追加・通過。テスト結果を更新。
- [x] Tier 3: `-j` JSONフィールド指定検索 — JSONの特定キーのみを対象にマッチ。テスト追加・通過。テスト結果を更新。

#### gf-find（ファイル検索）

- [x] Tier 1: `cmd/gf-find/` 作成（go.mod初期化、go.workに追加）。`-name` パターンによる再帰的ファイル検索。テスト作成・通過。テスト結果を `docs/test-results/gf-find.md` に記録。
- [x] Tier 2: `-type f/d`（ファイル/ディレクトリ）、`-size`（サイズ条件）、`-mtime`（更新日条件）。テスト追加・通過。テスト結果を更新。
- [x] Tier 3: `-exec` の安全版（確認プロンプト付き）、glob対応。テスト追加・通過。テスト結果を更新。

#### gf-sort（ソート）

- [ ] Tier 1: `cmd/gf-sort/` 作成（go.mod初期化、go.workに追加）。辞書順ソート、stdin対応。テスト作成・通過。テスト結果を `docs/test-results/gf-sort.md` に記録。
- [ ] Tier 2: `-n` 数値ソート、`-r` 逆順、`-k` キー指定、`-u` 重複除去。テスト追加・通過。テスト結果を更新。
- [ ] Tier 3: `-t` デリミタ指定。テスト追加・通過。テスト結果を更新。

#### gf-uniq（重複除去）

- [ ] Tier 1: `cmd/gf-uniq/` 作成（go.mod初期化、go.workに追加）。隣接重複行の除去、stdin対応。テスト作成・通過。テスト結果を `docs/test-results/gf-uniq.md` に記録。
- [ ] Tier 2: `-c` 出現回数カウント、`-d` 重複行のみ表示、`-i` 大文字小文字無視。テスト追加・通過。テスト結果を更新。
- [ ] Tier 3: `--global` 非隣接重複も除去するモード。テスト追加・通過。テスト結果を更新。

#### gf-cut（フィールド切り出し）

- [ ] Tier 1: `cmd/gf-cut/` 作成（go.mod初期化、go.workに追加）。`-d` デリミタ指定 + `-f` フィールド番号指定、stdin対応。テスト作成・通過。テスト結果を `docs/test-results/gf-cut.md` に記録。
- [ ] Tier 2: `-c` 文字位置指定、フィールド範囲（`1-3`, `2-`）。テスト追加・通過。テスト結果を更新。
- [ ] Tier 3: `--csv` CSV対応モード（クォート内のデリミタを無視）。テスト追加・通過。テスト結果を更新。

---

### Wave 3: テキスト変換・高度な処理

#### gf-sed（ストリームエディタ）

- [ ] Tier 1: `cmd/gf-sed/` 作成（go.mod初期化、go.workに追加）。`s/pattern/replace/` 基本置換（1行に最初の1つ）、stdin対応。テスト作成・通過。テスト結果を `docs/test-results/gf-sed.md` に記録。
- [ ] Tier 2: `g` フラグ（全置換）、アドレス指定（行番号、`/pattern/`）、`-i` in-place編集。テスト追加・通過。テスト結果を更新。
- [ ] Tier 3: マルチバイト安全な置換（rune単位処理）。テスト追加・通過。テスト結果を更新。

#### gf-xargs（引数構築・実行）

- [ ] Tier 1: `cmd/gf-xargs/` 作成（go.mod初期化、go.workに追加）。stdinから読み取った値を引数としてコマンド実行。テスト作成・通過。テスト結果を `docs/test-results/gf-xargs.md` に記録。
- [ ] Tier 2: `-n` 最大引数数指定、`-P` 並列実行数指定。テスト追加・通過。テスト結果を更新。
- [ ] Tier 3: `-0` null区切り対応、`--dry-run` 実行せずコマンドを表示。テスト追加・通過。テスト結果を更新。

#### gf-diff（差分表示）

- [ ] Tier 1: `cmd/gf-diff/` 作成（go.mod初期化、go.workに追加）。2ファイルの行単位diff（Myers algorithm）。テスト作成・通過。テスト結果を `docs/test-results/gf-diff.md` に記録。
- [ ] Tier 2: unified diff format (`-u`) 出力。テスト追加・通過。テスト結果を更新。
- [ ] Tier 3: カラー出力（ターミナル検出）、`--word` 単語単位diff。テスト追加・通過。テスト結果を更新。

#### gf-tree（ディレクトリツリー）

- [ ] Tier 1: `cmd/gf-tree/` 作成（go.mod初期化、go.workに追加）。再帰的ディレクトリツリー描画（罫線文字使用）。テスト作成・通過。テスト結果を `docs/test-results/gf-tree.md` に記録。
- [ ] Tier 2: `-L` 深さ制限、`-I` 除外パターン。テスト追加・通過。テスト結果を更新。
- [ ] Tier 3: ファイルサイズ表示、`--du` ディレクトリサイズ集計。テスト追加・通過。テスト結果を更新。

#### gf-jq（JSONプロセッサ）

- [ ] Tier 1: `cmd/gf-jq/` 作成（go.mod初期化、go.workに追加）。`.key`、`.key.nested`、`.[0]` の基本パスアクセス、stdin対応。テスト作成・通過。テスト結果を `docs/test-results/gf-jq.md` に記録。
- [ ] Tier 2: パイプ `|`、配列操作 `.[]`、`length`。テスト追加・通過。テスト結果を更新。
- [ ] Tier 3: `select(条件)` フィルタ、`keys`、`values`。テスト追加・通過。テスト結果を更新。

#### gf-hexdump（バイナリダンプ）

- [ ] Tier 1: `cmd/gf-hexdump/` 作成（go.mod初期化、go.workに追加）。16バイトずつの16進ダンプ表示、stdin対応。テスト作成・通過。テスト結果を `docs/test-results/gf-hexdump.md` に記録。
- [ ] Tier 2: ASCII併記、`-s` オフセット指定、`-n` 読み取りバイト数制限。テスト追加・通過。テスト結果を更新。
- [ ] Tier 3: カラー出力（NULL, 印字可能, 制御文字で色分け）。テスト追加・通過。テスト結果を更新。

---

### Task Final: 統合テスト

- [ ] 全ツールのパイプチェーン連携テスト。例: `gf-find . -name "*.go" | gf-grep "func" | gf-wc -l`。結果を `docs/test-results/integration.md` に記録。

## 成功基準

- 16ツール全て実装完了
- 全ツールで `go test` がパス
- 各ツールがパイプチェーンで連携動作すること
- 自分の日常作業で最低3つは実際に使えるクオリティ

## 技術的制約

- Go 1.22+
- 外部依存ゼロ（標準ライブラリのみ）
- `internal/` に共通コードを置く場合は最小限に留める
- 各ツールは独立して `go build` 可能であること

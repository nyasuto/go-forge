# GoForge — CLAUDE.md

## プロジェクト概要

UNIXコマンドをGoで再実装する学習プロジェクト。Ralph Loopによる自律量産開発。

## ディレクトリ構成

```
goforge/
├── CLAUDE.md          # このファイル（プロジェクトルール + サイン）
├── prd.md             # 要件定義 + タスクリスト（チェックボックス）
├── progress.md        # 完了タスクのログ
├── prompt.md          # Ralph Loop用プロンプト
├── ralph.sh           # Ralph Loop実行スクリプト
├── Makefile           # ビルド・テスト・品質チェック
├── go.work            # Go workspace
└── cmd/
    └── gf-xxx/        # 各ツール（独立モジュール）
        ├── go.mod
        ├── main.go
        └── main_test.go
```

## ビルド・テスト

```bash
# 特定ツールのテスト
cd cmd/gf-xxx && go test -v ./...

# 特定ツールのビルド
cd cmd/gf-xxx && go build -o gf-xxx .

# 全ツールテスト（go.work経由）
go test ./cmd/...
```

## コミットポリシー

- mainブランチに直接コミットしてよい
- Conventional Commits形式: `feat(gf-xxx): Tier N 実装`
- 1タスク完了ごとに1コミット
- `git add .` でOK

## 実装ルール

- 外部依存ゼロ（標準ライブラリのみ）
- `flag` パッケージでCLI引数パース
- テーブルドリブンテスト必須（正常系3+、異常系2+、エッジケース2+）
- stdin/stdout パイプ対応
- エラーはstderr、終了コード: 成功=0, エラー=1, 使用法エラー=2
- 引数なし or `-` → stdin読み取り

## 新しいツールを追加する手順

1. `cmd/gf-xxx/` ディレクトリ作成
2. `go mod init gf-xxx` で独立モジュール初期化
3. `main.go` + `main_test.go` 作成
4. `go.work` に `./cmd/gf-xxx` を追加
5. 実装 → テスト → コミット

---

## サイン（次回の自分への注意書き）

<!-- Ralph Loopの各ループで学んだことをここに追記する -->

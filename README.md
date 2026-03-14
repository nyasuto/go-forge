# GoForge

UNIXコマンドをGoでゼロから再実装するプロジェクト。標準ライブラリのみ使用、外部依存ゼロ。

## ツール一覧

| ツール | 説明 |
|--------|------|
| `gf-cat` | ファイル結合・表示（行番号付き表示対応） |
| `gf-head` | 先頭行の表示 |
| `gf-tail` | 末尾行の表示 |
| `gf-wc` | 行・単語・バイト数カウント |
| `gf-grep` | パターン検索（正規表現対応） |
| `gf-sort` | 行ソート（数値・逆順・重複除去） |
| `gf-uniq` | 重複行の除去・カウント |
| `gf-cut` | フィールド・文字位置の切り出し |
| `gf-sed` | ストリームエディタ（置換・削除） |
| `gf-find` | ファイル検索（名前・型・サイズ） |
| `gf-xargs` | 標準入力からコマンド実行 |
| `gf-tee` | 出力の分岐（stdout + ファイル） |
| `gf-diff` | ファイル差分（Myers algorithm、カラー・単語diff） |
| `gf-tree` | ディレクトリツリー表示（サイズ集計対応） |
| `gf-jq` | JSONプロセッサ（パス・パイプ・select・keys/values） |
| `gf-hexdump` | バイナリ16進ダンプ（カラー色分け） |
| `gf-claude-quota` | Claude Code使用量モニター（statusLine統合） |

全ツールがstdin/stdoutパイプに対応し、UNIXパイプラインで連携動作します。

## ビルド・テスト

```bash
# 全ツールをビルド
make build

# 全ツールのテスト実行
make test

# テスト + 静的解析
make quality

# 個別ツールのビルド
cd cmd/gf-cat && go build -o gf-cat .
```

## 要件

- Go 1.22+
- 外部依存なし（標準ライブラリのみ）

## プロジェクト構成

```
goforge/
├── Makefile
├── go.work
└── cmd/
    └── gf-xxx/          # 各ツール（独立Goモジュール）
        ├── go.mod
        ├── main.go
        └── main_test.go
```

各ツールは独立したGoモジュールとして実装されており、個別に `go build` 可能です。

## 使用例

```bash
# ファイル検索 → パターン抽出 → 行数カウント
gf-find . -name "*.go" | gf-xargs gf-grep "func" | gf-wc -l

# 頻度ランキング
gf-cat access.log | gf-cut -d' ' -f1 | gf-sort | gf-uniq -c | gf-sort -rn | gf-head -n 10

# JSON処理
gf-cat data.json | gf-jq '.users[] | select(.age > 20) | .name'

# ディレクトリサイズ集計
gf-tree --du -L 2 ./src

# バイナリ解析
gf-hexdump -n 64 --color binary.dat
```

## ライセンス

MIT

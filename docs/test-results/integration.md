# Integration Test Results — パイプチェーン連携テスト

実行日: 2026-03-14

## テスト概要

全16ツール（gf-claude-quota除く）のパイプチェーン連携を検証するシェルスクリプトベースの統合テスト。

## テスト結果

```
PASS: 21 / FAIL: 0 / TOTAL: 21
```

## テスト一覧

| # | パイプチェーン | 検証内容 | 結果 |
|---|---------------|---------|------|
| 1 | gf-find \| gf-xargs \| gf-grep \| gf-wc | .goファイル検索→func行カウント | PASS |
| 2 | gf-cat \| gf-sort \| gf-uniq | ファイル読み込み→ソート→重複除去 | PASS |
| 3 | gf-cat \| gf-sort \| gf-uniq -c \| gf-sort -n -r \| gf-head | 頻度ランキング（top 3） | PASS |
| 4 | gf-cat \| gf-grep \| gf-wc | ログERROR行カウント | PASS |
| 5 | gf-cat \| gf-tail \| gf-cut \| gf-sort \| gf-uniq | CSV都市列抽出→ソート→重複除去 | PASS |
| 6 | gf-cat \| gf-sed \| gf-grep \| gf-wc | sed置換後のgrep検索 | PASS |
| 7 | gf-cat \| gf-head \| gf-tail | 特定行の抽出（3行目） | PASS |
| 8 | gf-cat \| gf-sort -n -u \| gf-head \| gf-tail | 数値ソート→中央値付近取得 | PASS |
| 9a | gf-cat \| gf-tee \| gf-wc | tee分岐→stdout行数カウント | PASS |
| 9b | gf-wc (tee output) | tee書き出しファイルの行数確認 | PASS |
| 10 | gf-cat \| gf-jq \| gf-grep | JSON名前展開→grepフィルタ | PASS |
| 11 | gf-cat \| gf-tail \| gf-cut \| gf-grep | CSV名前列抽出→grepフィルタ | PASS |
| 12 | gf-find \| gf-sort \| gf-head | ファイル検索→ソート→先頭N件 | PASS |
| 13 | echo \| gf-sed \| gf-sort \| gf-uniq -c \| gf-sort -n -r \| gf-head | sed置換→ソート→カウント→ランキング | PASS |
| 14 | echo \| gf-hexdump \| gf-head | hexdump出力→先頭行取得 | PASS |
| 15 | gf-cat \| gf-jq \| gf-sort \| gf-uniq -c \| gf-sort -n -r \| gf-head | JSON都市集計ランキング | PASS |
| 16 | gf-cat \| gf-grep -v \| gf-sed \| gf-wc | grep反転→sed置換→行数カウント | PASS |
| 17 | gf-cat (multi) \| gf-sort \| gf-uniq \| gf-wc | 複数ファイル結合→ソート→重複除去→カウント | PASS |
| 18 | gf-diff -u \| gf-grep \| gf-grep -v \| gf-wc | unified diff→変更行フィルタ | PASS |
| 19 | gf-tree \| gf-grep \| gf-wc | ツリー出力→.goファイル検索 | PASS |
| 20 | echo \| gf-xargs -n 1 \| gf-sort | xargs分割→ソート | PASS |

## 使用ツール一覧

全16ツールがパイプチェーンで連携動作することを確認:

- gf-cat, gf-head, gf-tail, gf-wc, gf-tee
- gf-grep, gf-find, gf-sort, gf-uniq, gf-cut
- gf-sed, gf-xargs, gf-diff, gf-tree, gf-jq, gf-hexdump

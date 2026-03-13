# gf-diff テスト結果

## Tier 1: 2ファイルの行単位diff（Myers algorithm）

実行日: 2026-03-14

### テスト概要

| カテゴリ | テスト数 | 結果 |
|----------|----------|------|
| 単体テスト (myersDiff) | 9 | ALL PASS |
| 単体テスト (readLinesFromString) | 3 | ALL PASS |
| 統合テスト (run) | 7 | ALL PASS |
| エラー系テスト | 4 | ALL PASS |
| ファイル未検出テスト | 2 | ALL PASS |
| バージョン表示テスト | 1 | ALL PASS |
| 大量入力テスト | 1 | ALL PASS |
| **合計** | **28** | **ALL PASS** |

### テストケース詳細

#### myersDiff 単体テスト
- identical files: 同一内容 → 全opEqual、差分なし
- insert one line: 1行挿入 → opInsert検出
- delete one line: 1行削除 → opDelete検出
- replace one line: 1行置換 → opDelete+opInsert
- both empty: 空×空 → editsなし
- first empty: 空 vs 2行 → 全opInsert
- second empty: 2行 vs 空 → 全opDelete
- completely different: 完全不一致 → delete+insert
- multibyte lines: マルチバイト行の差分検出

#### 統合テスト
- identical files returns 0: 同一ファイル → exit 0
- different files returns 1 with diff output: 差分あり → exit 1 + diff出力
- insert lines: 行追加の差分表示
- delete lines: 行削除の差分表示
- empty first file: 空ファイル vs 内容あり
- empty second file: 内容あり vs 空ファイル
- multibyte content: マルチバイト内容の差分

#### エラー系
- no args: 引数なし → exit 2
- one arg: 引数1つ → exit 2
- three args: 引数3つ → exit 2
- unknown flag: 不明フラグ → exit 2

### 実行結果

```
PASS
ok  	gf-diff	0.332s
```

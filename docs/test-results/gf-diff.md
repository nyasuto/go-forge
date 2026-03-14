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

---

## Tier 2: unified diff format (`-u`) 出力

実行日: 2026-03-14

### テスト概要

| カテゴリ | テスト数 | 結果 |
|----------|----------|------|
| 単体テスト (buildHunks) | 6 | ALL PASS |
| 統合テスト (unified format) | 8 | ALL PASS |
| **Tier 2 追加分** | **14** | **ALL PASS** |
| **累計** | **42** | **ALL PASS** |

### テストケース詳細

#### buildHunks 単体テスト
- single change with context: 単一変更 → コンテキスト付き1 hunk
- two distant changes become two hunks: 離れた2変更 → 2 hunks
- insert at beginning: 先頭挿入 → 正しいhunkヘッダ
- delete at end: 末尾削除 → 正しいhunkヘッダ
- zero context: コンテキスト0 → 変更行のみ
- nearby changes merge into single hunk: 近接変更 → 1 hunkにマージ

#### unified format 統合テスト
- basic unified diff: 基本的なunified出力（---/+++/@@ヘッダ、-/+行）
- identical files no output: 同一ファイル → 出力なし、exit 0
- insert lines unified: 行挿入のunified表示
- delete lines unified: 行削除のunified表示
- multibyte unified: マルチバイト文字のunified表示
- file headers contain filenames: ファイル名がヘッダに含まれる
- empty first file unified: 空ファイルからの挿入
- empty second file unified: 全行削除

---

## Tier 3: カラー出力・`--word` 単語単位diff

実行日: 2026-03-14

### テスト概要

| カテゴリ | テスト数 | 結果 |
|----------|----------|------|
| 単体テスト (splitWords) | 8 | ALL PASS |
| 単体テスト (wordDiffLine) | 6 | ALL PASS |
| 統合テスト (color always) | 2 | ALL PASS |
| 統合テスト (color never) | 1 | ALL PASS |
| 統合テスト (color auto) | 1 | ALL PASS |
| 統合テスト (color invalid) | 1 | ALL PASS |
| 統合テスト (word diff) | 6 | ALL PASS |
| 大量入力テスト | 1 | ALL PASS |
| **Tier 3 追加分** | **26** | **ALL PASS** |
| **累計** | **76** | **ALL PASS** |

### テストケース詳細

#### splitWords 単体テスト
- empty: 空文字列 → nil
- single word: 単一単語
- two words: 2単語（スペース区切り）
- leading space: 先頭スペース
- trailing space: 末尾スペース
- multiple spaces: 複数スペース保持
- tabs and spaces: タブ+スペース混在
- multibyte words: マルチバイト単語

#### wordDiffLine 単体テスト
- single word change: 単語置換 → `[-old-]`/`[+new+]`マーカー
- word insertion: 単語挿入 → `[+word+]`マーカー
- word deletion: 単語削除 → `[-word-]`マーカー
- completely different: 全単語異なる
- identical lines: 同一行 → マーカーなし
- multibyte word change: マルチバイト単語の差分

#### カラー出力 統合テスト
- normal mode with color: `--color=always` で赤/緑/リセットコード出力
- unified mode with color: `--color=always` で太字赤/太字緑/シアンヘッダ
- color never: `--color=never` でANSIコードなし
- color auto (non-terminal): bytes.Buffer → カラーなし
- color invalid: 不正値 → exit 2

#### word diff 統合テスト
- normal word diff: 通常モードで`[-/-]`/`[+/+]`マーカー表示
- unified word diff: `-u --word`でunified形式+ワードマーカー
- word diff with color: `--word --color=always` 併用
- multibyte word diff: マルチバイト単語の差分マーカー
- word insert only: 単語挿入のみ
- word delete only: 単語削除のみ

### 実行結果

```
PASS
ok  	gf-diff	0.341s
```

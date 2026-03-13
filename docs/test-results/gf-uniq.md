# gf-uniq テスト結果

## Tier 1: コア機能（隣接重複行の除去、stdin対応）

### 実行日: 2026-03-14

### 単体テスト (TestProcessReader): 7件

| # | テストケース | 結果 |
|---|------------|------|
| 1 | adjacent duplicates removed | PASS |
| 2 | no duplicates | PASS |
| 3 | all same lines | PASS |
| 4 | empty input | PASS |
| 5 | single line | PASS |
| 6 | multibyte characters | PASS |
| 7 | empty lines as duplicates | PASS |

### 単体テスト (TestRun): 7件

| # | テストケース | 結果 |
|---|------------|------|
| 1 | stdin input | PASS |
| 2 | stdin via hyphen | PASS |
| 3 | file input | PASS |
| 4 | nonexistent file | PASS |
| 5 | mixed: valid file and nonexistent | PASS |
| 6 | large repeated input | PASS |
| 7 | non-adjacent duplicates preserved | PASS |

### 統合テスト (TestIntegration): 8件

| # | テストケース | 結果 |
|---|------------|------|
| 1 | basic stdin dedup | PASS |
| 2 | file dedup | PASS |
| 3 | version flag | PASS |
| 4 | nonexistent file | PASS |
| 5 | empty stdin | PASS |
| 6 | pipe: echo with duplicates | PASS |
| 7 | multibyte stdin | PASS |
| 8 | non-adjacent not removed | PASS |

### 結果サマリ
- 全22件 ALL PASS
- 正常系: 隣接重複除去、stdin対応、ファイル入力、複数ファイル
- 異常系: 存在しないファイル、有効+無効ファイル混在
- エッジケース: 空入力、大量重複行(10000行)、マルチバイト文字、空行の重複、非隣接重複の保持

---

## Tier 2: -c 出現回数カウント、-d 重複行のみ表示、-i 大文字小文字無視

### 実行日: 2026-03-14

### 単体テスト (TestProcessReader) 追加: 14件

| # | テストケース | 結果 |
|---|------------|------|
| 8 | count: adjacent duplicates | PASS |
| 9 | count: no duplicates | PASS |
| 10 | count: single line | PASS |
| 11 | duplicates: show only duplicated lines | PASS |
| 12 | duplicates: no duplicates means no output | PASS |
| 13 | duplicates: all same | PASS |
| 14 | ignore case: adjacent case-different duplicates | PASS |
| 15 | ignore case: no match without -i | PASS |
| 16 | ignore case: multibyte | PASS |
| 17 | count + duplicates | PASS |
| 18 | count + ignore case | PASS |
| 19 | duplicates + ignore case | PASS |
| 20 | all three options | PASS |
| 21 | empty input with options | PASS |

### 単体テスト (TestRun) 追加: 3件

| # | テストケース | 結果 |
|---|------------|------|
| 8 | count option via run | PASS |
| 9 | duplicates option via run | PASS |
| 10 | ignore case option via run | PASS |

### 統合テスト (TestIntegration) 追加: 9件

| # | テストケース | 結果 |
|---|------------|------|
| 9 | -c count flag | PASS |
| 10 | -d duplicates only flag | PASS |
| 11 | -i case insensitive flag | PASS |
| 12 | -c -d combined | PASS |
| 13 | -c -i combined | PASS |
| 14 | -d -i combined | PASS |
| 15 | -c -d -i all combined | PASS |
| 16 | -c with file input | PASS |
| 17 | -d with empty input | PASS |

### 結果サマリ
- 累計48件 ALL PASS（単体21件 + run10件 + 統合17件）
- -c: 出現回数を`%7d`フォーマットでプレフィックス表示
- -d: 2回以上出現した行のみ出力
- -i: `strings.EqualFold`による大文字小文字無視比較
- 全オプション組み合わせ（-c -d、-c -i、-d -i、-c -d -i）動作確認済み

---

## Tier 3: --global 非隣接重複除去モード

### 実行日: 2026-03-14

### 単体テスト (TestProcessReader) 追加: 12件

| # | テストケース | 結果 |
|---|------------|------|
| 22 | global: non-adjacent duplicates removed | PASS |
| 23 | global: all unique | PASS |
| 24 | global: all same | PASS |
| 25 | global: empty input | PASS |
| 26 | global: single line | PASS |
| 27 | global: multibyte | PASS |
| 28 | global: preserves first occurrence order | PASS |
| 29 | global + count | PASS |
| 30 | global + duplicates | PASS |
| 31 | global + ignore case | PASS |
| 32 | global + count + duplicates | PASS |
| 33 | global + count + duplicates + ignore case | PASS |

### 統合テスト (TestIntegration) 追加: 9件

| # | テストケース | 結果 |
|---|------------|------|
| 18 | --global removes non-adjacent duplicates | PASS |
| 19 | --global with file input | PASS |
| 20 | --global -c count | PASS |
| 21 | --global -d duplicates only | PASS |
| 22 | --global -i case insensitive | PASS |
| 23 | --global -c -d -i all combined | PASS |
| 24 | --global empty input | PASS |
| 25 | --global multibyte | PASS |
| 26 | --global large input | PASS |

### 結果サマリ
- 累計69件 ALL PASS（単体33件 + run10件 + 統合26件）
- --global: mapで全行を追跡し、非隣接重複も除去
- 初出順序を保持（出現順にスライスで管理）
- -c/-d/-i との全組み合わせ動作確認済み
- 大量入力（10000行）でも正常動作

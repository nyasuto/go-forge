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

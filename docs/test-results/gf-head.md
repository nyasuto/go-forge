# gf-head テスト結果

## Tier 1: コア機能

### 実行日: 2026-03-14

### 単体テスト (TestHead): 9件 全PASS

| # | テストケース | 種別 | 結果 |
|---|-------------|------|------|
| 1 | default 10 lines from 15 line input | 正常系 | PASS |
| 2 | fewer lines than n | 正常系 | PASS |
| 3 | custom n=3 | 正常系 | PASS |
| 4 | n=1 | 正常系 | PASS |
| 5 | empty input | エッジケース | PASS |
| 6 | n=0 outputs nothing | エッジケース | PASS |
| 7 | multibyte characters | エッジケース | PASS |
| 8 | lines without trailing newline | エッジケース | PASS |
| 9 | blank lines | エッジケース | PASS |

### 統合テスト (TestIntegration): 8件 全PASS

| # | テストケース | 種別 | 結果 |
|---|-------------|------|------|
| 1 | stdin default 10 lines | 正常系 | PASS |
| 2 | file argument | 正常系 | PASS |
| 3 | nonexistent file exits 1 | 異常系 | PASS |
| 4 | version flag | 正常系 | PASS |
| 5 | stdin with hyphen | 正常系 | PASS |
| 6 | multiple files with headers | 正常系 | PASS |
| 7 | empty file | エッジケース | PASS |
| 8 | pipe: echo \| gf-head | 正常系 | PASS |

## Tier 2: -c バイト数指定

### 実行日: 2026-03-14

### 単体テスト (TestHeadBytes): 7件 全PASS

| # | テストケース | 種別 | 結果 |
|---|-------------|------|------|
| 1 | first 5 bytes | 正常系 | PASS |
| 2 | more bytes than input | 正常系 | PASS |
| 3 | exact length | 正常系 | PASS |
| 4 | zero bytes | エッジケース | PASS |
| 5 | empty input | エッジケース | PASS |
| 6 | multibyte characters truncated at byte boundary | エッジケース | PASS |
| 7 | newlines counted as bytes | エッジケース | PASS |

### 統合テスト (TestIntegration Tier 2): 4件 全PASS

| # | テストケース | 種別 | 結果 |
|---|-------------|------|------|
| 1 | -c bytes from stdin | 正常系 | PASS |
| 2 | -c bytes from file | 正常系 | PASS |
| 3 | -c with multiple files | 正常系 | PASS |
| 4 | -c more bytes than file | エッジケース | PASS |

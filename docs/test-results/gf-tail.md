# gf-tail テスト結果

## 実行日: 2026-03-14

## Tier 1: コア機能

### 単体テスト (TestTail): 11件 全PASS

| テスト名 | 種別 | 結果 |
|----------|------|------|
| last 10 lines from 15 line input | 正常系 | PASS |
| fewer lines than n | 正常系 | PASS |
| custom n=3 | 正常系 | PASS |
| n=1 | 正常系 | PASS |
| n=0 outputs nothing | 異常系 | PASS |
| empty input | エッジケース | PASS |
| multibyte characters | エッジケース | PASS |
| lines without trailing newline | エッジケース | PASS |
| blank lines | エッジケース | PASS |
| exact n lines | エッジケース | PASS |
| single line | エッジケース | PASS |

### 統合テスト (TestIntegration): 8件 全PASS

| テスト名 | 結果 |
|----------|------|
| stdin default 10 lines | PASS |
| file argument | PASS |
| nonexistent file exits 1 | PASS |
| version flag | PASS |
| stdin with hyphen | PASS |
| multiple files with headers | PASS |
| empty file | PASS |
| pipe: echo \| gf-tail | PASS |

## Tier 2: -f フォローモード

### 統合テスト: 5件 全PASS

| テスト名 | 種別 | 結果 |
|----------|------|------|
| -f follows appended data | 正常系 | PASS |
| -f without file exits 2 | 異常系 | PASS |
| -f with stdin hyphen exits 2 | 異常系 | PASS |
| -f with multiple files exits 2 | 異常系 | PASS |
| -f detects file truncation | エッジケース | PASS |

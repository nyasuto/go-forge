# gf-sed テスト結果

## Tier 1: 基本置換・stdin対応

- 実行日: 2026-03-14
- 結果: ALL PASS

### テストケース一覧

#### TestParseExpression (11件)
| # | テスト名 | 種別 | 結果 |
|---|---------|------|------|
| 1 | basic substitution | 正常系 | PASS |
| 2 | without trailing delimiter | 正常系 | PASS |
| 3 | regex pattern | 正常系 | PASS |
| 4 | empty replacement | 正常系 | PASS |
| 5 | custom delimiter | 正常系 | PASS |
| 6 | escaped delimiter in pattern | エッジ | PASS |
| 7 | replacement with backslash | エッジ | PASS |
| 8 | unknown command | 異常系 | PASS |
| 9 | invalid regex | 異常系 | PASS |
| 10 | no pattern or replacement | 異常系 | PASS |
| 11 | empty expression after s | 異常系 | PASS |

#### TestApplySubstitution (10件)
| # | テスト名 | 種別 | 結果 |
|---|---------|------|------|
| 1 | basic replacement | 正常系 | PASS |
| 2 | first match only | 正常系 | PASS |
| 3 | no match | 正常系 | PASS |
| 4 | regex replacement | 正常系 | PASS |
| 5 | delete pattern | 正常系 | PASS |
| 6 | replace at beginning | 正常系 | PASS |
| 7 | replace at end | 正常系 | PASS |
| 8 | capture group | エッジ | PASS |
| 9 | multibyte characters | エッジ | PASS |
| 10 | empty line | エッジ | PASS |

#### TestSplitByDelim (4件)
| # | テスト名 | 種別 | 結果 |
|---|---------|------|------|
| 1 | basic split | 正常系 | PASS |
| 2 | escaped delimiter | エッジ | PASS |
| 3 | no trailing delimiter | 正常系 | PASS |
| 4 | pipe delimiter | 正常系 | PASS |

#### TestRun (10件)
| # | テスト名 | 種別 | 結果 |
|---|---------|------|------|
| 1 | stdin basic substitution | 正常系 | PASS |
| 2 | file input | 正常系 | PASS |
| 3 | multiple files | 正常系 | PASS |
| 4 | stdin with hyphen | 正常系 | PASS |
| 5 | no match passes through | 正常系 | PASS |
| 6 | empty input | エッジ | PASS |
| 7 | multibyte replacement | エッジ | PASS |
| 8 | first match only per line | 正常系 | PASS |
| 9 | nonexistent file | 異常系 | PASS |
| 10 | mixed existing and nonexistent files | 異常系 | PASS |

#### TestVersion (1件)
| # | テスト名 | 種別 | 結果 |
|---|---------|------|------|
| 1 | version check | 正常系 | PASS |

### 合計: 36件 ALL PASS

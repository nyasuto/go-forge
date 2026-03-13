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

## Tier 2: g フラグ・アドレス指定・-i in-place編集

- 実行日: 2026-03-14
- 結果: ALL PASS

### テストケース一覧

#### TestParseExpressionTier2 (9件)
| # | テスト名 | 種別 | 結果 |
|---|---------|------|------|
| 1 | g flag | 正常系 | PASS |
| 2 | no g flag | 正常系 | PASS |
| 3 | unknown flag | 異常系 | PASS |
| 4 | line number address | 正常系 | PASS |
| 5 | last line address | 正常系 | PASS |
| 6 | pattern address | 正常系 | PASS |
| 7 | line address with g flag | 正常系 | PASS |
| 8 | unterminated address regex | 異常系 | PASS |
| 9 | invalid address regex | 異常系 | PASS |

#### TestGlobalFlag (5件)
| # | テスト名 | 種別 | 結果 |
|---|---------|------|------|
| 1 | g replaces all occurrences | 正常系 | PASS |
| 2 | g with regex replaces all | 正常系 | PASS |
| 3 | g with no match | 正常系 | PASS |
| 4 | g with overlapping-like matches | エッジ | PASS |
| 5 | g with empty match pattern | 正常系 | PASS |

#### TestAddressRun (9件)
| # | テスト名 | 種別 | 結果 |
|---|---------|------|------|
| 1 | line number address | 正常系 | PASS |
| 2 | line 1 address | 正常系 | PASS |
| 3 | last line address | 正常系 | PASS |
| 4 | last line address single line | エッジ | PASS |
| 5 | pattern address | 正常系 | PASS |
| 6 | pattern address with g flag | 正常系 | PASS |
| 7 | line address with g flag | 正常系 | PASS |
| 8 | address beyond line count | エッジ | PASS |
| 9 | pattern address no match lines | 正常系 | PASS |

#### TestInPlace (7件)
| # | テスト名 | 種別 | 結果 |
|---|---------|------|------|
| 1 | basic in-place edit | 正常系 | PASS |
| 2 | in-place with g flag | 正常系 | PASS |
| 3 | in-place multiple files | 正常系 | PASS |
| 4 | in-place preserves file permissions | エッジ | PASS |
| 5 | in-place nonexistent file | 異常系 | PASS |
| 6 | in-place stdin rejected | 異常系 | PASS |
| 7 | in-place with address | 正常系 | PASS |

#### TestGlobalFlagRun (3件)
| # | テスト名 | 種別 | 結果 |
|---|---------|------|------|
| 1 | global replace multiple lines | 正常系 | PASS |
| 2 | global with capture groups | 正常系 | PASS |
| 3 | global with multibyte | エッジ | PASS |

### Tier 2 追加: 33件、累計: 62件 ALL PASS

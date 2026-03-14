# gf-jq テスト結果

## Tier 1: 基本パスアクセス・stdin対応

- 実行日: 2026-03-14
- 結果: ALL PASS

### テスト内訳

#### TestParseFilter (12件)
- identity (`.`)
- single key (`.name`)
- nested key (`.user.name`)
- array index (`.[0]`)
- key then index (`.items.[0]`)
- key then index no dot (`.items[0]`)
- deep nesting (`.a.b.c.d`)
- index then key (`.[0].name`)
- no leading dot → error
- unclosed bracket → error
- non-numeric index → error
- empty key → error

#### TestApplyFilter (20件)
- identity returns full object
- simple key access
- numeric value
- nested key access
- array index
- array first element
- key then array index
- nested object in array
- boolean value
- null value
- missing key returns null
- out of range index returns null
- negative index
- deeply nested
- multibyte key (日本語)
- float value
- empty object
- empty array
- nested array access
- string with special chars

#### TestApplyFilterErrors (2件)
- key access on array → error
- index access on object → error

#### TestInvalidJSON (1件)
- invalid JSON → exit 1 + stderr

#### TestEmptyInput (1件)
- empty input → exit 1

#### TestRun (5件)
- version flag
- missing filter → exit 2
- stdin input
- stdin with hyphen
- invalid filter → exit 2

#### TestRunWithFile (1件)
- file input with nested access

#### TestRunWithNonExistentFile (1件)
- nonexistent file → exit 1

#### TestRunMultipleFiles (1件)
- multiple file inputs

#### TestLargeJSON (1件)
- deeply nested value access

### 合計: 45件 ALL PASS

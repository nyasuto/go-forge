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

### 小計: 45件 ALL PASS

---

## Tier 2: パイプ `|`・配列操作 `.[]`・`length`

- 実行日: 2026-03-14
- 結果: ALL PASS

### テスト内訳

#### TestParseFilter 追加 (14件 → 合計26件)
- iterator (`.[]`)
- key then iterator (`.items[]`)
- iterator then key (`.[].name`)
- pipe two stages (`.a | .b`)
- pipe three stages (`.a | .b | .c`)
- pipe with iterator (`.[] | .name`)
- pipe with identity (`. | .name`)
- length function (`length`)
- pipe to length (`. | length`)
- key pipe length (`.items | length`)
- empty pipe stage leading → error
- empty pipe stage trailing → error
- empty pipe stage middle → error
- (既存エラーケース1件)

#### TestIterator (12件)
- iterate array
- iterate string array
- iterate object values sorted by key
- iterate nested array via key
- iterate then access key
- iterate with pipe
- key pipe iterate pipe key
- empty array iteration
- empty object iteration
- iterate array of arrays
- nested iterate (`.[] | .[]`)
- iterate multibyte values

#### TestIteratorErrors (4件)
- iterate over string → error
- iterate over number → error
- iterate over boolean → error
- iterate over null → error

#### TestLength (13件)
- length of array
- length of empty array
- length of object
- length of empty object
- length of string
- length of multibyte string (Unicode rune単位)
- length of null → 0
- length of number (absolute value)
- length of positive number
- length via pipe
- length of each element (`.[] | length`)
- length of nested array via key
- length of string with emoji

#### TestLengthErrors (1件)
- length of boolean → error

#### TestPipe (5件)
- identity pipe key
- key pipe key
- three stage pipe
- pipe fan out then collect
- pipe with length after iterate

#### TestRun 追加 (3件 → 合計8件)
- pipe via stdin
- length via stdin
- empty pipe stage error → exit 2

#### TestUnknownFunction (1件)
- unknown bare word → parse error

### 小計: 30件追加

### 累計: 75件 ALL PASS

---

## Tier 3: `select(条件)` フィルタ・`keys`・`values`

- 実行日: 2026-03-14
- 結果: ALL PASS

### テスト内訳

#### TestKeys (6件)
- keys of object（ソート済みキー配列）
- keys of empty object
- keys of array（インデックス配列）
- keys of empty array
- keys via pipe
- keys with multibyte keys

#### TestKeysErrors (3件)
- keys of null → error
- keys of string → error
- keys of number → error

#### TestValues (4件)
- values of object sorted by key
- values of empty object
- values of array
- values via pipe

#### TestValuesErrors (2件)
- values of null → error
- values of string → error

#### TestSelect (15件)
- select equal number (`select(.age == 30)`)
- select greater than (`select(.v > 2)`)
- select less than (`select(.v < 4)`)
- select greater equal (`select(.v >= 2)`)
- select less equal (`select(.v <= 2)`)
- select not equal (`select(.v != 2)`)
- select string equal (`select(.name == "alice")`)
- select truthiness true（`.active`がtrueの要素のみ）
- select truthiness null excluded（null値を除外）
- select none match（マッチなし→出力なし）
- select all match（全マッチ→全出力）
- select with nested key（ネストされたキーでフィルタ）
- select null comparison（`select(.v == null)`)
- select then access key（selectの後にキーアクセス）
- select with multibyte string（日本語文字列比較）

#### TestSelectErrors (2件)
- empty select → parse error
- invalid value in select → parse error

#### TestParseFilterTier3 (7件)
- keys function
- values function
- pipe to keys
- pipe to values
- select basic
- select with pipe
- empty select error

### 小計: 39件追加

### 累計: 114件 ALL PASS

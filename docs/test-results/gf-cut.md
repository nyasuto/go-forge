# gf-cut テスト結果

## Tier 1: コア機能

- 実行日: 2026-03-14
- 結果: ALL PASS (32件)

### テスト内訳

#### TestParseFields (12件)
- single field: PASS
- multiple fields: PASS
- range: PASS
- open end range: PASS
- open start range: PASS
- mixed: PASS
- invalid field zero: PASS
- invalid field negative: PASS
- invalid decreasing range: PASS
- invalid empty: PASS
- invalid non-numeric: PASS
- invalid dash-dash: PASS

#### TestSelectFields (6件)
- single field: PASS
- multiple fields: PASS
- range: PASS
- open end range: PASS
- field out of range: PASS
- partial out of range: PASS

#### TestRun (9件)
- tab delimiter default field 2: PASS
- comma delimiter field 1 and 3: PASS
- field range 2-3: PASS
- open end range 3-: PASS
- field beyond columns outputs empty: PASS
- empty input: PASS
- multibyte delimiter and content: PASS
- single column input: PASS
- no delimiter in line outputs whole line as field 1: PASS

#### TestRunWithFiles (5件)
- single file: PASS
- multiple files: PASS
- nonexistent file: PASS
- stdin via hyphen: PASS
- mixed file and stdin: PASS

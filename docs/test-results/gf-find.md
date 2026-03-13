# gf-find テスト結果

## Tier 1: コア機能

### 実行日: 2026-03-14

### テスト結果: ALL PASS

### テストケース一覧

| # | テスト名 | 種別 | 結果 |
|---|---------|------|------|
| 1 | matchName/empty_pattern_matches_anything | 正常系 | PASS |
| 2 | matchName/exact_match | 正常系 | PASS |
| 3 | matchName/glob_star | 正常系 | PASS |
| 4 | matchName/glob_no_match | 正常系 | PASS |
| 5 | matchName/question_mark | 正常系 | PASS |
| 6 | matchName/multibyte_name | エッジケース | PASS |
| 7 | matchName/invalid_pattern | 異常系 | PASS |
| 8 | Integration/all_files_in_tree_(no_-name) | 正常系 | PASS |
| 9 | Integration/find_txt_files | 正常系 | PASS |
| 10 | Integration/find_go_files | 正常系 | PASS |
| 11 | Integration/find_specific_file | 正常系 | PASS |
| 12 | Integration/no_matches | エッジケース | PASS |
| 13 | Integration/multibyte_filename_match | エッジケース | PASS |
| 14 | Integration/nonexistent_path | 異常系 | PASS |
| 15 | Integration/multiple_paths | 正常系 | PASS |
| 16 | Integration/version_flag | 正常系 | PASS |
| 17 | FindEmptyDir | エッジケース | PASS |
| 18 | FindSingleFile | エッジケース | PASS |

- 単体テスト: 7件
- 統合テスト: 11件
- 合計: 18件 ALL PASS

## Tier 2: -type f/d、-size、-mtime オプション

### 実行日: 2026-03-14

### テスト結果: ALL PASS

### テストケース一覧

| # | テスト名 | 種別 | 結果 |
|---|---------|------|------|
| 19 | ParseSizeExpr/bytes | 正常系 | PASS |
| 20 | ParseSizeExpr/kilobytes | 正常系 | PASS |
| 21 | ParseSizeExpr/megabytes | 正常系 | PASS |
| 22 | ParseSizeExpr/gigabytes | 正常系 | PASS |
| 23 | ParseSizeExpr/blocks_default | 正常系 | PASS |
| 24 | ParseSizeExpr/greater_than | 正常系 | PASS |
| 25 | ParseSizeExpr/less_than | 正常系 | PASS |
| 26 | ParseSizeExpr/empty | 異常系 | PASS |
| 27 | ParseSizeExpr/only_sign | 異常系 | PASS |
| 28 | ParseSizeExpr/only_unit | 異常系 | PASS |
| 29 | ParseSizeExpr/invalid_unit | 異常系 | PASS |
| 30 | ParseSizeExpr/not_a_number | 異常系 | PASS |
| 31 | ParseMtimeExpr/exact_days | 正常系 | PASS |
| 32 | ParseMtimeExpr/more_than | 正常系 | PASS |
| 33 | ParseMtimeExpr/less_than | 正常系 | PASS |
| 34 | ParseMtimeExpr/zero_days | 正常系 | PASS |
| 35 | ParseMtimeExpr/empty | 異常系 | PASS |
| 36 | ParseMtimeExpr/only_sign | 異常系 | PASS |
| 37 | ParseMtimeExpr/not_a_number | 異常系 | PASS |
| 38 | MatchType/empty_filter_matches_file | 正常系 | PASS |
| 39 | MatchType/empty_filter_matches_dir | 正常系 | PASS |
| 40 | MatchType/f_matches_file | 正常系 | PASS |
| 41 | MatchType/f_rejects_dir | 正常系 | PASS |
| 42 | MatchType/d_matches_dir | 正常系 | PASS |
| 43 | MatchType/d_rejects_file | 正常系 | PASS |
| 44 | TypeFilter/type_f_-_files_only | 正常系 | PASS |
| 45 | TypeFilter/type_d_-_directories_only | 正常系 | PASS |
| 46 | TypeFilter/type_f_with_name_filter | 正常系 | PASS |
| 47 | TypeFilter/invalid_type_value | 異常系 | PASS |
| 48 | SizeFilter/size_greater_than_500c | 正常系 | PASS |
| 49 | SizeFilter/size_less_than_100c | 正常系 | PASS |
| 50 | SizeFilter/size_exact_1k | 正常系 | PASS |
| 51 | SizeFilter/size_greater_than_5k | 正常系 | PASS |
| 52 | SizeFilter/invalid_size_expression | 異常系 | PASS |
| 53 | MtimeFilter/mtime_-5 | 正常系 | PASS |
| 54 | MtimeFilter/mtime_+7 | 正常系 | PASS |
| 55 | MtimeFilter/mtime_+30 | 正常系 | PASS |
| 56 | MtimeFilter/mtime_0 | 正常系 | PASS |
| 57 | MtimeFilter/invalid_mtime_expression | 異常系 | PASS |
| 58 | CombinedFilters/type_f_+_name_+_size | 正常系 | PASS |
| 59 | CombinedFilters/type_d_only | 正常系 | PASS |
| 60 | NoMatchWithFilter | エッジケース | PASS |

- Tier 2 追加 単体テスト: 25件（parseSizeExpr 12件 + parseMtimeExpr 7件 + matchType 6件）
- Tier 2 追加 統合テスト: 17件
- 累計: 60件 ALL PASS

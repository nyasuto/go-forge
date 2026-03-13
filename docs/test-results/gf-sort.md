# gf-sort テスト結果

## Tier 1: コア機能

### 実行日: 2026-03-14

### テスト結果: ALL PASS (17件)

#### 単体テスト (4件)
| テスト名 | 内容 | 結果 |
|----------|------|------|
| ReadLinesFrom/normal_lines | 通常の複数行読み取り | PASS |
| ReadLinesFrom/empty_input | 空入力 | PASS |
| ReadLinesFrom/single_line_no_newline | 改行なし単一行 | PASS |
| ReadLinesFrom/multibyte | マルチバイト文字 | PASS |

#### 統合テスト (13件)
| テスト名 | 内容 | 結果 |
|----------|------|------|
| BasicSort/sort_from_stdin | stdin辞書順ソート | PASS |
| BasicSort/already_sorted | ソート済み入力 | PASS |
| BasicSort/reverse_order_input | 逆順入力 | PASS |
| BasicSort/empty_input | 空入力 | PASS |
| BasicSort/single_line | 単一行 | PASS |
| BasicSort/multibyte_sort | マルチバイトソート | PASS |
| BasicSort/case_sensitivity | 大文字小文字の順序 | PASS |
| BasicSort/duplicate_lines | 重複行の保持 | PASS |
| BasicSort/version_flag | --version表示 | PASS |
| FileInput/single_file | 単一ファイル入力 | PASS |
| FileInput/multiple_files_merged_and_sorted | 複数ファイル結合ソート | PASS |
| FileInput/nonexistent_file | 存在しないファイル→exit 1 | PASS |
| FileInput/stdin_via_hyphen | ハイフンでstdin読み取り | PASS |

## Tier 2: -n 数値ソート、-r 逆順、-k キー指定、-u 重複除去

### 実行日: 2026-03-14

### テスト結果: ALL PASS (34件追加、累計51件)

#### 単体テスト (18件追加)
| テスト名 | 内容 | 結果 |
|----------|------|------|
| ExtractKey/no_key_field | キー指定なし→行全体 | PASS |
| ExtractKey/field_1 | フィールド1抽出 | PASS |
| ExtractKey/field_2 | フィールド2抽出 | PASS |
| ExtractKey/field_3 | フィールド3抽出 | PASS |
| ExtractKey/field_out_of_range | 範囲外→空文字 | PASS |
| ExtractKey/multiple_spaces | 複数スペース区切り | PASS |
| ExtractKey/tabs | タブ区切り | PASS |
| ParseNumber/integer | 整数パース | PASS |
| ParseNumber/negative | 負数パース | PASS |
| ParseNumber/float | 浮動小数点パース | PASS |
| ParseNumber/non-numeric | 非数値→0 | PASS |
| ParseNumber/empty | 空文字→0 | PASS |
| ParseNumber/leading_spaces | 前後スペース付き数値 | PASS |
| Dedup/no_duplicates | 重複なし | PASS |
| Dedup/consecutive_duplicates | 連続重複除去 | PASS |
| Dedup/all_same | 全て同一 | PASS |
| Dedup/empty | 空スライス | PASS |
| Dedup/single | 単一要素 | PASS |

#### 統合テスト (16件追加)
| テスト名 | 内容 | 結果 |
|----------|------|------|
| Tier2Options/numeric_sort_-n | 数値ソート | PASS |
| Tier2Options/numeric_sort_with_non-numeric_lines | 非数値行混在の数値ソート | PASS |
| Tier2Options/numeric_sort_negative_numbers | 負数の数値ソート | PASS |
| Tier2Options/numeric_sort_floats | 浮動小数点数値ソート | PASS |
| Tier2Options/reverse_sort_-r | 逆順ソート | PASS |
| Tier2Options/reverse_numeric_sort_-n_-r | 逆順数値ソート | PASS |
| Tier2Options/unique_-u | 重複除去 | PASS |
| Tier2Options/unique_numeric_-n_-u | 数値ソート+重複除去 | PASS |
| Tier2Options/key_field_-k_2 | フィールド2でソート | PASS |
| Tier2Options/key_field_numeric_-k_2_-n | フィールド2で数値ソート | PASS |
| Tier2Options/key_field_with_reverse_-k_1_-r | フィールド1で逆順ソート | PASS |
| Tier2Options/key_out_of_range_treated_as_empty | 範囲外キー→空文字扱い | PASS |
| Tier2Options/unique_with_reverse_-u_-r | 逆順+重複除去 | PASS |
| Tier2Options/all_options_combined_-k_2_-n_-r_-u | 全オプション組み合わせ | PASS |
| Tier2Options/empty_input_with_options | 空入力+オプション | PASS |
| Tier2Options/multibyte_with_reverse | マルチバイト逆順ソート | PASS |

## Tier 3: -t デリミタ指定

### 実行日: 2026-03-14

### テスト結果: ALL PASS (14件追加、累計58件)

#### 単体テスト (7件追加)
| テスト名 | 内容 | 結果 |
|----------|------|------|
| ExtractKey/comma_delimiter_field_1 | カンマ区切りフィールド1 | PASS |
| ExtractKey/comma_delimiter_field_2 | カンマ区切りフィールド2 | PASS |
| ExtractKey/colon_delimiter | コロン区切り | PASS |
| ExtractKey/tab_delimiter | タブ区切り | PASS |
| ExtractKey/delimiter_field_out_of_range | デリミタ指定で範囲外→空文字 | PASS |
| ExtractKey/delimiter_with_empty_fields | デリミタ指定で空フィールド | PASS |
| ExtractKey/pipe_delimiter | パイプ区切り | PASS |

#### 統合テスト (7件追加)
| テスト名 | 内容 | 結果 |
|----------|------|------|
| Tier2Options/delimiter_-t_comma_with_-k | カンマ区切り+キー指定 | PASS |
| Tier2Options/delimiter_-t_colon_with_-k_-n | コロン区切り+数値ソート | PASS |
| Tier2Options/delimiter_-t_pipe_with_-k_-r | パイプ区切り+逆順 | PASS |
| Tier2Options/delimiter_-t_tab_with_-k | タブ区切り+キー指定 | PASS |
| Tier2Options/delimiter_-t_with_-k_-u | デリミタ+重複除去 | PASS |
| Tier2Options/delimiter_-t_with_empty_fields | 空フィールド含むデリミタ指定 | PASS |
| Tier2Options/delimiter_-t_without_-k_sorts_whole_line | -tのみ指定（-kなし）→行全体ソート | PASS |

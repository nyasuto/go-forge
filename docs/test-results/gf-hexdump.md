# gf-hexdump テスト結果

## Tier 1: 16進ダンプ表示・stdin対応

実行日: 2026-03-14

### テスト結果: ALL PASS (23件)

```
=== RUN   TestFormatLine
=== RUN   TestFormatLine/full_16_bytes
=== RUN   TestFormatLine/partial_line
=== RUN   TestFormatLine/non-printable_bytes
=== RUN   TestFormatLine/offset_at_boundary
=== RUN   TestFormatLine/8_bytes_boundary_gap
--- PASS: TestFormatLine (5件)

=== RUN   TestHexdump
=== RUN   TestHexdump/simple_ASCII
=== RUN   TestHexdump/empty_input
=== RUN   TestHexdump/exactly_16_bytes
=== RUN   TestHexdump/more_than_16_bytes
=== RUN   TestHexdump/binary_data_with_nulls
=== RUN   TestHexdump/all_byte_values_0x20-0x2f
=== RUN   TestHexdump/multibyte_UTF-8
--- PASS: TestHexdump (7件)

=== RUN   TestRun
=== RUN   TestRun/stdin_no_args
=== RUN   TestRun/stdin_with_dash
=== RUN   TestRun/file_argument
=== RUN   TestRun/multiple_files
=== RUN   TestRun/nonexistent_file
=== RUN   TestRun/version_flag
=== RUN   TestRun/unknown_flag
=== RUN   TestRun/nonexistent_file_with_valid_file
=== RUN   TestRun/large_input_crossing_multiple_lines
--- PASS: TestRun (9件)

=== RUN   TestHexdumpExactly16Bytes
--- PASS: TestHexdumpExactly16Bytes (1件)

=== RUN   TestHexdump17Bytes
--- PASS: TestHexdump17Bytes (1件)
```

### テストカバレッジ

- **正常系**: ASCII入力、16バイト完全行、複数行、バイナリデータ、マルチバイトUTF-8
- **異常系**: 存在しないファイル、不正フラグ
- **エッジケース**: 空入力、17バイト（行境界超え）、非印字文字、オフセット境界

## Tier 2: -s オフセット指定・-n バイト数制限

実行日: 2026-03-14

### テスト結果: ALL PASS (累計40件、追加17件)

```
=== RUN   TestHexdumpSkip
=== RUN   TestHexdumpSkip/skip_4_bytes
=== RUN   TestHexdumpSkip/skip_past_end
=== RUN   TestHexdumpSkip/skip_0_is_noop
=== RUN   TestHexdumpSkip/skip_16_to_second_line
--- PASS: TestHexdumpSkip (4件)

=== RUN   TestHexdumpLimit
=== RUN   TestHexdumpLimit/limit_5_bytes
=== RUN   TestHexdumpLimit/limit_0_bytes
=== RUN   TestHexdumpLimit/limit_larger_than_input
=== RUN   TestHexdumpLimit/limit_exactly_16
--- PASS: TestHexdumpLimit (4件)

=== RUN   TestHexdumpSkipAndLimit
=== RUN   TestHexdumpSkipAndLimit/skip_5_limit_5
=== RUN   TestHexdumpSkipAndLimit/skip_and_limit_to_single_byte
--- PASS: TestHexdumpSkipAndLimit (2件)

=== RUN   TestRunWithSkipAndLimit
=== RUN   TestRunWithSkipAndLimit/skip_flag_with_file
=== RUN   TestRunWithSkipAndLimit/limit_flag_with_file
=== RUN   TestRunWithSkipAndLimit/skip_and_limit_with_file
=== RUN   TestRunWithSkipAndLimit/skip_with_stdin
=== RUN   TestRunWithSkipAndLimit/limit_with_stdin
=== RUN   TestRunWithSkipAndLimit/negative_skip
=== RUN   TestRunWithSkipAndLimit/negative_limit
--- PASS: TestRunWithSkipAndLimit (7件)
```

### テストカバレッジ（Tier 2追加分）

- **正常系**: skip 4バイト、skip 16バイト（行境界）、limit 5バイト、limit 16バイト、skip+limit組み合わせ、ファイル・stdinでのフラグ動作
- **異常系**: 負のskip値（exit 2）、負のlimit値（exit 2）
- **エッジケース**: skip 0（noop）、skipが入力超過（空出力）、limit 0（空出力）、limitが入力より大（全出力）、skip+limitで1バイトのみ

## Tier 3: カラー出力（NULL, 印字可能, 制御文字で色分け）

実行日: 2026-03-14

### テスト結果: ALL PASS (累計62件、追加22件)

```
=== RUN   TestByteColor
=== RUN   TestByteColor/null_byte
=== RUN   TestByteColor/printable_A
=== RUN   TestByteColor/printable_space
=== RUN   TestByteColor/printable_tilde
=== RUN   TestByteColor/control_tab
=== RUN   TestByteColor/control_newline
=== RUN   TestByteColor/control_0x01
=== RUN   TestByteColor/control_0x1f
=== RUN   TestByteColor/control_DEL
=== RUN   TestByteColor/high_byte_0x80
=== RUN   TestByteColor/high_byte_0xff
--- PASS: TestByteColor (11件)

=== RUN   TestFormatLineColor
=== RUN   TestFormatLineColor/null_bytes_get_dim_color
=== RUN   TestFormatLineColor/printable_bytes_get_green
=== RUN   TestFormatLineColor/control_bytes_get_red
=== RUN   TestFormatLineColor/high_bytes_get_blue
=== RUN   TestFormatLineColor/mixed_bytes
--- PASS: TestFormatLineColor (5件)

=== RUN   TestColorNoColor
--- PASS: TestColorNoColor (1件)

=== RUN   TestRunColorFlag
=== RUN   TestRunColorFlag/color_always
=== RUN   TestRunColorFlag/color_never
=== RUN   TestRunColorFlag/color_auto_with_non-terminal
=== RUN   TestRunColorFlag/invalid_color_mode
--- PASS: TestRunColorFlag (4件)

=== RUN   TestHexdumpExactly16Bytes (既存1件)
=== RUN   TestHexdump17Bytes (既存1件)
```

### テストカバレッジ（Tier 3追加分）

- **正常系**: NULL→dim、印字可能→緑、制御文字→赤、高バイト→青の色分け、混合バイトの色分け
- **異常系**: 不正なカラーモード（exit 2）
- **エッジケース**: color=false時にエスケープシーケンスが含まれないこと、auto時に非ターミナルでカラー無効

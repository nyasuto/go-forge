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

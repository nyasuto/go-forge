# gf-tee テスト結果

## Tier 1: コア機能

### 実行日: 2026-03-14

### テスト件数: 10件（テーブルドリブン8件 + 個別テスト2件）

### 結果: 全PASS

```
=== RUN   TestRun
=== RUN   TestRun/stdin_to_stdout_only_(no_files)
=== RUN   TestRun/stdin_to_stdout_and_one_file
=== RUN   TestRun/multiline_input
=== RUN   TestRun/cannot_create_file_in_nonexistent_directory
=== RUN   TestRun/unknown_flag
=== RUN   TestRun/empty_input
=== RUN   TestRun/multibyte_input
=== RUN   TestRun/large_input
--- PASS: TestRun (0.01s)
=== RUN   TestVersion
--- PASS: TestVersion (0.00s)
=== RUN   TestMultipleFiles
--- PASS: TestMultipleFiles (0.00s)
PASS
ok  	gf-tee	0.402s
```

### カバレッジ

- 正常系: 3件（stdout のみ、1ファイル、複数行）
- 異常系: 2件（存在しないディレクトリ、不正フラグ）
- エッジケース: 3件（空入力、マルチバイト、大量入力）
- 個別テスト: 2件（バージョン表示、複数ファイル同時書き出し）

## Tier 2: -a appendモード

### 実行日: 2026-03-14

### テスト件数: 16件（Tier 1: 10件 + Tier 2追加: 6件）

### 結果: 全PASS

```
=== RUN   TestRun
=== RUN   TestRun/stdin_to_stdout_only_(no_files)
=== RUN   TestRun/stdin_to_stdout_and_one_file
=== RUN   TestRun/multiline_input
=== RUN   TestRun/cannot_create_file_in_nonexistent_directory
=== RUN   TestRun/unknown_flag
=== RUN   TestRun/empty_input
=== RUN   TestRun/multibyte_input
=== RUN   TestRun/large_input
--- PASS: TestRun (0.01s)
=== RUN   TestAppendMode
=== RUN   TestAppendMode/append_to_existing_file
=== RUN   TestAppendMode/append_to_empty_file
=== RUN   TestAppendMode/append_creates_file_if_not_exists
=== RUN   TestAppendMode/append_multibyte_content
=== RUN   TestAppendMode/append_empty_input_to_existing_file
--- PASS: TestAppendMode (0.01s)
=== RUN   TestAppendWithoutFlag
--- PASS: TestAppendWithoutFlag (0.00s)
=== RUN   TestVersion
--- PASS: TestVersion (0.00s)
=== RUN   TestMultipleFiles
--- PASS: TestMultipleFiles (0.00s)
PASS
ok  	gf-tee	0.416s
```

### カバレッジ（Tier 2追加分）

- 正常系: 3件（既存ファイルに追記、空ファイルに追記、ファイル新規作成）
- エッジケース: 2件（マルチバイト追記、空入力で既存内容保持）
- 対照テスト: 1件（-aなしで上書き確認）

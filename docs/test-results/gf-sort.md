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

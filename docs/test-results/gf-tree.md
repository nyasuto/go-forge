# gf-tree テスト結果

## 実行日: 2026-03-14

## テスト概要

| カテゴリ | テスト数 | 結果 |
|----------|----------|------|
| 単体テスト (walkDir) | 7 | ALL PASS |
| 単体テスト (printTree) | 3 | ALL PASS |
| 統合テスト | 8 | ALL PASS |
| **合計** | **18** | **ALL PASS** |

## 単体テスト: walkDir

| テスト名 | 内容 | 結果 |
|----------|------|------|
| simple_directory_with_files | ファイルのみのディレクトリ | PASS |
| nested_directories | ネストされたディレクトリ構造 | PASS |
| empty_directory | 空ディレクトリ | PASS |
| directory_with_mixed_content | ディレクトリとファイル混在 | PASS |
| multibyte_filenames | 日本語・絵文字ファイル名 | PASS |
| alphabetical_sorting | アルファベット順ソート | PASS |
| deep_nesting_with_connectors | 深いネストのコネクタ文字 | PASS |

## 単体テスト: printTree

| テスト名 | 内容 | 結果 |
|----------|------|------|
| prints_root_directory_name | ルートディレクトリ名の出力 | PASS |
| error_on_non-existent_path | 存在しないパスでエラー | PASS |
| error_on_file_path | ファイルパスでエラー | PASS |

## 統合テスト

| テスト名 | 内容 | 結果 |
|----------|------|------|
| basic_tree_output | 基本ツリー出力 | PASS |
| default_to_current_directory | デフォルトでカレントディレクトリ | PASS |
| multiple_directories | 複数ディレクトリ指定 | PASS |
| non-existent_directory | 存在しないディレクトリ→exit 1 | PASS |
| version_flag | --version表示 | PASS |
| empty_directory | 空ディレクトリ→0 directories, 0 files | PASS |
| correct_directory_and_file_counts | ディレクトリ・ファイル数の正確性 | PASS |
| tree_structure_ordering | アルファベット順出力 | PASS |

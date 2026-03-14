# gf-tree テスト結果

## 実行日: 2026-03-14

## テスト概要

| カテゴリ | テスト数 | 結果 |
|----------|----------|------|
| 単体テスト (walkDir) | 7 | ALL PASS |
| 単体テスト (printTree) | 3 | ALL PASS |
| 単体テスト (walkDir -L 深さ制限) | 4 | ALL PASS |
| 単体テスト (walkDir -I 除外パターン) | 5 | ALL PASS |
| 単体テスト (isExcluded) | 5 | ALL PASS |
| 統合テスト | 14 | ALL PASS |
| **合計** | **38** | **ALL PASS** |

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

## 単体テスト: walkDir -L 深さ制限

| テスト名 | 内容 | 結果 |
|----------|------|------|
| depth_1_shows_only_top-level | -L 1でトップレベルのみ表示 | PASS |
| depth_2_shows_two_levels | -L 2で2階層表示 | PASS |
| depth_0_means_unlimited | -L 0で無制限 | PASS |
| depth_exceeding_tree_depth_shows_all | 深い-Lは全表示 | PASS |

## 単体テスト: walkDir -I 除外パターン

| テスト名 | 内容 | 結果 |
|----------|------|------|
| exclude_by_extension | 拡張子で除外（*.txt） | PASS |
| exclude_directory_by_name | ディレクトリ名で除外 | PASS |
| exclude_with_glob_pattern | globパターンで除外（file*） | PASS |
| no_match_excludes_nothing | マッチなしで除外なし | PASS |
| exclude_applied_at_all_levels | 全階層で除外適用 | PASS |

## 単体テスト: isExcluded

| テスト名 | 内容 | 結果 |
|----------|------|------|
| match_extension | 拡張子マッチ | PASS |
| no_match_extension | 拡張子不一致 | PASS |
| match_prefix | プレフィックスマッチ | PASS |
| empty_pattern | 空パターン→除外なし | PASS |
| exact_match | 完全一致 | PASS |

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
| -L_depth_limit | -L 1でネスト表示抑制 | PASS |
| -L_2_depth_limit | -L 2で3階層目を抑制 | PASS |
| -I_exclude_pattern | -I *.txtで.txtファイル除外 | PASS |
| -I_exclude_directory | -I dir1でディレクトリ除外 | PASS |
| -L_and_-I_combined | -Lと-Iの組み合わせ | PASS |
| -L_negative_value_exits_with_code_2 | -L負値→exit 2 | PASS |
| tree_structure_ordering | アルファベット順出力 | PASS |

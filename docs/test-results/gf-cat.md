# gf-cat テスト結果

## Tier 1

- 実行日: 2026-03-14
- 結果: **ALL PASS**

### テストケース一覧

#### 単体テスト (TestCat)

| テスト名 | 種別 | 結果 |
|-----------|------|------|
| 単一行 | 正常系 | PASS |
| 複数行 | 正常系 | PASS |
| 改行なし末尾 | 正常系 | PASS |
| 空入力 | エッジケース | PASS |
| マルチバイト文字 | エッジケース | PASS |
| 大きな入力 | エッジケース | PASS |
| 空行のみ | エッジケース | PASS |

#### 統合テスト (TestCatFile)

| テスト名 | 種別 | 結果 |
|-----------|------|------|
| 単一ファイル | 正常系 | PASS |
| 複数ファイル連結 | 正常系 | PASS |
| stdinから読み取り（引数なし） | 正常系 | PASS |
| ハイフンでstdin | 正常系 | PASS |
| ファイルとstdinの混合 | 正常系 | PASS |
| 存在しないファイル | 異常系 | PASS |
| 存在しないファイルと存在するファイル | 異常系 | PASS |
| バージョン表示 | 正常系 | PASS |

## Tier 2

- 実行日: 2026-03-14
- 結果: **ALL PASS**

### テストケース一覧

#### 単体テスト (TestCatNumberLines) — `-n` 行番号表示

| テスト名 | 種別 | 結果 |
|-----------|------|------|
| 単一行に行番号 | 正常系 | PASS |
| 複数行に行番号 | 正常系 | PASS |
| 空行にも行番号 | 正常系 | PASS |
| 空入力 | エッジケース | PASS |
| マルチバイト文字に行番号 | エッジケース | PASS |

#### 単体テスト (TestCatSqueezeBlank) — `-s` 連続空行圧縮

| テスト名 | 種別 | 結果 |
|-----------|------|------|
| 連続空行を圧縮 | 正常系 | PASS |
| 空行なしはそのまま | 正常系 | PASS |
| 先頭の連続空行を圧縮 | 正常系 | PASS |
| 空入力 | エッジケース | PASS |
| 全て空行 | エッジケース | PASS |
| 空行1行はそのまま | エッジケース | PASS |

#### 単体テスト (TestCatNumberAndSqueeze) — `-n -s` 組み合わせ

| テスト名 | 種別 | 結果 |
|-----------|------|------|
| 行番号+圧縮の組み合わせ | 正常系 | PASS |
| 圧縮後の行番号は連番 | 正常系 | PASS |

#### 統合テスト (TestCatFile) — Tier 2追加分

| テスト名 | 種別 | 結果 |
|-----------|------|------|
| -nで行番号表示 | 正常系 | PASS |
| -nで複数ファイル連番 | 正常系 | PASS |
| -sで連続空行圧縮 | 正常系 | PASS |
| -n -sの組み合わせ | 正常系 | PASS |

## Tier 3

- 実行日: 2026-03-14
- 結果: **ALL PASS**

### テストケース一覧

#### 単体テスト (TestHighlightLine) — シンタックスハイライト

| テスト名 | 種別 | 結果 |
|-----------|------|------|
| Goキーワード | 正常系 | PASS |
| Go文字列リテラル | 正常系 | PASS |
| Goコメント | 正常系 | PASS |
| Go数値 | 正常系 | PASS |
| Go行内コメント | 正常系 | PASS |
| Pythonキーワード | 正常系 | PASS |
| Pythonハッシュコメント | 正常系 | PASS |
| Python文字列内のハッシュ | 正常系 | PASS |
| JSキーワード | 正常系 | PASS |
| JSONキーと値 | 正常系 | PASS |
| JSON数値 | 正常系 | PASS |
| JSONブール | 正常系 | PASS |
| JSONnull | 正常系 | PASS |
| YAMLキーと値 | 正常系 | PASS |
| YAMLコメント | 正常系 | PASS |
| YAMLブール値 | 正常系 | PASS |
| YAML数値 | 正常系 | PASS |
| YAML文字列値 | 正常系 | PASS |
| 空行 | エッジケース | PASS |
| マルチバイト文字列 | エッジケース | PASS |
| yml拡張子 | エッジケース | PASS |

#### 単体テスト (TestDetectLanguage) — 拡張子検出

| テスト名 | 種別 | 結果 |
|-----------|------|------|
| main.go | 正常系 | PASS |
| script.py | 正常系 | PASS |
| app.js | 正常系 | PASS |
| config.json | 正常系 | PASS |
| config.yaml | 正常系 | PASS |
| config.yml | 正常系 | PASS |
| readme.txt | 異常系 | PASS |
| noext | 異常系 | PASS |
| FILE.GO | エッジケース | PASS |
| path/to/main.go | エッジケース | PASS |

#### 単体テスト (TestCatWithHighlight) — ハイライト付きcat

| テスト名 | 種別 | 結果 |
|-----------|------|------|
| color=always でGoコード | 正常系 | PASS |

#### 単体テスト (TestCatColorNever) — ハイライト無効

| テスト名 | 種別 | 結果 |
|-----------|------|------|
| color=never でGoコード | 正常系 | PASS |

#### 統合テスト (TestCatFile) — Tier 3追加分

| テスト名 | 種別 | 結果 |
|-----------|------|------|
| --color=always でGoファイル | 正常系 | PASS |
| --color=always でJSONファイル | 正常系 | PASS |
| --color=never でGoファイル | 正常系 | PASS |
| --color=always でtxtファイル（ハイライトなし） | エッジケース | PASS |
| --color=always -n でGoファイル | 正常系 | PASS |

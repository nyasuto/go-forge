# gf-xargs テスト結果

## Tier 1: コア機能

**実行日**: 2026-03-14
**結果**: ALL PASS (30件)

### テスト内訳

#### 単体テスト: splitArgs (8件)
- simple words
- multiple spaces
- tabs
- double quotes
- single quotes
- empty string
- only spaces
- mixed quotes

#### 単体テスト: run (9件)
- default echo command（コマンド未指定→echo）
- explicit command（grep -l TODO + stdin引数）
- empty stdin（実行なし）
- command failure（エラー→exit 1）
- version flag
- multiple lines（複数行→全引数結合）
- quoted args in stdin（クォート内スペース保持）
- multiple words per line（行内複数単語分割）
- unknown flag（→exit 2）

#### エッジケーステスト (4件)
- multibyte input（日本語引数）
- lines with only whitespace（空白行→実行なし）
- large input（1000引数）
- error message on stderr（エラー出力確認）

#### 統合テスト (9件)
- echo with real executor
- echo multiple items
- command not found（→exit 1）
- version output
- empty stdin no execution
- stdin pipe with printf
- default echo no command specified
- multibyte echo（マルチバイト文字のecho）

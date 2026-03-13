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

## Tier 2: -n 最大引数数指定・-P 並列実行数指定

**実行日**: 2026-03-14
**結果**: ALL PASS (累計48件)

### 追加テスト内訳

#### 単体テスト: splitBatches (6件)
- n=0 single batch（全アイテム1バッチ）
- n=1（1個ずつ分割）
- n=2 even（偶数個の均等分割）
- n=2 odd（奇数個の端数バッチ）
- n=5 larger than items（アイテム数より大きいn）
- n=3 exact（ちょうど割り切れる）

#### 単体テスト: runWithN (5件)
- n=1 splits into individual calls（1個ずつ実行）
- n=2 splits into batches（2個ずつバッチ実行）
- n=0 means all in one call（全引数1回実行）
- n with extra command args（コマンド追加引数との組み合わせ）
- negative n（負の値→exit 2）

#### 単体テスト: runWithP (5件)
- P=2 parallel execution（2並列実行）
- P=4 more workers than batches（ワーカー数>バッチ数）
- P=0 invalid（0→exit 2）
- P with error propagation（エラー伝播）
- P=1 sequential with n（逐次+バッチ分割）

#### 統合テスト追加 (3件)
- n=1 real echo（実echoで1個ずつ実行）
- n=2 real echo batches（実echoで2個ずつバッチ）
- P=2 real parallel echo（実echoで2並列、順序不問で全出力確認）

## Tier 3: -0 null区切り対応・--dry-run コマンド表示

**実行日**: 2026-03-14
**結果**: ALL PASS (累計69件)

### 追加テスト内訳

#### 単体テスト: readItemsNull (8件)
- simple null-separated（基本null区切り）
- no trailing null（末尾nullなし）
- empty items skipped（空アイテムスキップ）
- spaces preserved（スペース保持）
- newlines in items（アイテム内改行保持）
- empty input（空入力）
- multibyte（マルチバイト文字）
- paths with spaces（スペース含むパス）

#### 単体テスト: shellJoin (5件)
- simple（基本結合）
- with spaces（スペース含む引数→クォート）
- with quotes（シングルクォート含む引数→エスケープ）
- empty arg（空引数→''）
- special chars（特殊文字含むパス）

#### 単体テスト: runWithNullDelim (3件)
- null delim basic（-0基本動作、スペース保持）
- null delim with newlines（-0で改行含むアイテム）
- null delim with -n（-0と-nの組み合わせ）

#### 単体テスト: runDryRun (5件)
- dry-run basic（コマンド表示のみ、実行なし）
- dry-run with -n（バッチ分割表示）
- dry-run with spaces in args（スペース含む引数のクォート表示）
- dry-run empty stdin（空入力→出力なし）
- dry-run with -0（null区切りとの組み合わせ）

#### 統合テスト追加 (4件)
- null delim real echo（実echoでnull区切り入力）
- dry-run real（実行環境でdry-run確認）
- dry-run with n=1（実行環境でバッチ分割dry-run）
- null delim with -n=1 real（-0と-n=1の組み合わせ実行）

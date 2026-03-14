# GoForge 進捗ログ

<!-- 各タスク完了時に以下の形式で追記する -->
<!-- ## Task名 -->
<!-- - 完了日: YYYY-MM-DD -->
<!-- - 作業内容: やったことの要約 -->

## Task 0: プロジェクト初期化
- 完了日: 2026-03-14
- 作業内容: go.work初期化、Makefile作成（build/test/qualityターゲット）、docs/test-results/ディレクトリ作成。全ターゲットの動作確認済み。

## gf-cat Tier 1: コア機能
- 完了日: 2026-03-14
- 作業内容: cmd/gf-cat/ 作成（go.mod初期化、go.workに追加）。ファイル連結表示、stdin対応（引数なし・ハイフン）、複数ファイル引数対応、--version表示、エラーハンドリング（存在しないファイル→exit 1）。単体テスト7件+統合テスト8件、全PASS。

## gf-cat Tier 2: 行番号表示・連続空行圧縮
- 完了日: 2026-03-14
- 作業内容: `-n` 行番号表示（`%6d\t` フォーマット、複数ファイルで連番継続）、`-s` 連続空行圧縮、`-n -s` 組み合わせ対応。オプション未指定時はio.Copyによる高速パス維持。単体テスト20件+統合テスト12件、全PASS。

## gf-cat Tier 3: シンタックスハイライト自動検出
- 完了日: 2026-03-14
- 作業内容: `--color` フラグ（auto/always/never）によるシンタックスハイライト。ファイル拡張子（.go, .py, .js, .json, .yaml, .yml）に基づく言語自動検出。キーワード・文字列・コメント・数値・JSONキー・YAMLキーのANSIカラー出力。文字列内コメントマーカーの誤検出防止。autoモードはターミナル検出+既知拡張子で自動有効化。highlight.goに分離して実装。単体テスト21件+統合テスト5件追加、全PASS。

## gf-head Tier 1: コア機能
- 完了日: 2026-03-14
- 作業内容: cmd/gf-head/ 作成（go.mod初期化、go.workに追加）。デフォルト10行表示、`-n` 行数指定、stdin対応（引数なし・ハイフン）、複数ファイル対応（ヘッダ付き）、--version表示、エラーハンドリング（存在しないファイル→exit 1）。単体テスト9件+統合テスト8件、全PASS。

## gf-head Tier 2: -c バイト数指定
- 完了日: 2026-03-14
- 作業内容: `-c` バイト数指定オプション追加。`io.CopyN`による効率的なバイト読み取り。`-c`指定時は行モードではなくバイトモードで動作。複数ファイル対応（ヘッダ付き）はTier 1で実装済み。単体テスト7件+統合テスト4件追加、全PASS。

## gf-head Tier 3: ストリーミングモード
- 完了日: 2026-03-14
- 作業内容: `-F` ストリーミングモード追加。stdinからN行受け取るたびにANSIエスケープ（クリア＆カーソルホーム）で画面クリアし再表示。`-c`との併用・ファイル引数との併用はエラー（exit 2）。端数行も最終バッチとして表示。単体テスト8件+統合テスト5件追加、全PASS。

## gf-tail Tier 1: コア機能
- 完了日: 2026-03-14
- 作業内容: cmd/gf-tail/ 作成（go.mod初期化、go.workに追加）。デフォルト末尾10行表示、`-n` 行数指定、stdin対応（引数なし・ハイフン）、複数ファイル対応（ヘッダ付き）、--version表示、エラーハンドリング（存在しないファイル→exit 1）。リングバッファによる効率的な末尾行保持。単体テスト11件+統合テスト8件、全PASS。

## gf-tail Tier 2: -f フォローモード
- 完了日: 2026-03-14
- 作業内容: `-f` フォローモード追加。ファイル末尾表示後、100msポーリングで追記を監視し新しいデータを出力し続ける。`os.Stat`でサイズ変更を検出、ファイルtruncation（サイズ縮小）にも対応してファイル先頭から再読み取り。`-f`はファイル引数1つのみ対応（stdin・ハイフン・複数ファイルはexit 2）。統合テスト5件追加、全PASS。

## gf-tail Tier 3: -p パターンハイライト
- 完了日: 2026-03-14
- 作業内容: `-p パターン` オプション追加。正規表現でマッチした部分をANSIカラー（太字赤）でハイライト表示。`-f`フォローモードとの組み合わせにも対応（行単位読み取りに切り替え）。不正な正規表現はexit 2。`highlightLine`関数で`FindAllStringIndex`を使い、マッチ部分のみ着色。単体テスト8件+統合テスト5件追加、全PASS。

## gf-wc Tier 1: コア機能
- 完了日: 2026-03-14
- 作業内容: cmd/gf-wc/ 作成（go.mod初期化、go.workに追加）。行数・単語数・バイト数カウント、`-l`（行数のみ）・`-w`（単語数のみ）・`-c`（バイト数のみ）フラグ、stdin対応（引数なし・ハイフン）、複数ファイル対応（合計行表示）、--version表示、エラーハンドリング（存在しないファイル→exit 1）。単体テスト12件+統合テスト8件、全PASS。

## gf-wc Tier 2: -m 文字数（rune対応）
- 完了日: 2026-03-14
- 作業内容: `-m` 文字数カウントオプション追加。`unicode/utf8.RuneCount`によるrune単位の文字数カウント。`-m`指定時は文字数のみ表示、フラグ未指定時はデフォルト表示（行数・単語数・バイト数）に文字数は含めない（`wc`互換）。countsstructにcharsフィールド追加。単体テスト2件（絵文字含む、混合マルチバイト複数行）+統合テスト4件（-m ASCII、-mマルチバイト、複数ファイル合計行、-m絵文字）追加、全26件PASS。

## gf-wc Tier 3: --json JSON形式出力
- 完了日: 2026-03-14
- 作業内容: `--json` フラグ追加。`encoding/json`による構造化JSON出力。単一ファイル/stdinの場合は`{lines, words, bytes, chars, file?}`のフラットオブジェクト、複数ファイルの場合は`{files: [...], total: {...}}`の階層構造。`omitempty`でstdin時のfileフィールド省略。統合テスト5件（stdin入力、ファイル入力、複数ファイル、マルチバイト、空入力）追加、全31件PASS。

## gf-tee Tier 1: コア機能
- 完了日: 2026-03-14
- 作業内容: cmd/gf-tee/ 作成（go.mod初期化、go.workに追加）。stdinを読み取りstdout＋指定ファイルに同時書き出し。`io.MultiWriter`で複数Writer統合。複数ファイル同時書き出し対応、--version表示、エラーハンドリング（存在しないディレクトリ→exit 1、不正フラグ→exit 2）。テスト10件（正常系3、異常系2、エッジケース3、個別テスト2）、全PASS。

## gf-tee Tier 2: -a appendモード
- 完了日: 2026-03-14
- 作業内容: `-a` appendモード追加。`os.OpenFile`で`O_APPEND|O_CREATE|O_WRONLY`フラグを使用し既存ファイルへの追記に対応。`-a`なしの場合は従来通り`os.Create`で上書き。テスト6件追加（既存ファイル追記、空ファイル追記、新規作成、マルチバイト追記、空入力保持、-aなし上書き確認）、全16件PASS。

## gf-tee Tier 3: 複数ファイル同時書き出し・--ts タイムスタンプ付与
- 完了日: 2026-03-14
- 作業内容: `--ts` タイムスタンプ付与オプション追加。`bufio.Scanner`による行単位読み取りに切り替え、各行に`[2006-01-02T15:04:05.000Z07:00]`形式のISO 8601タイムスタンプをプレフィックス付与。`-a`や複数ファイルとの組み合わせにも対応。テスト用に`nowFunc`変数でtime.Now差し替え可能。テスト9件追加（タイムスタンプ7件+複数ファイルappend1件）、全25件PASS。

## gf-grep Tier 1: コア機能
- 完了日: 2026-03-14
- 作業内容: cmd/gf-grep/ 作成（go.mod初期化、go.workに追加）。固定文字列マッチ（`strings.Contains`）、マッチ行出力、stdin対応（引数なし・ハイフン）、複数ファイル対応（ファイル名プレフィックス付き）、--version表示、エラーハンドリング（存在しないファイル→stderr、パターンなし→exit 2、マッチなし→exit 1）。単体テスト9件+統合テスト8件、全17件PASS。

## gf-grep Tier 2: 正規表現対応・オプション拡張
- 完了日: 2026-03-14
- 作業内容: `regexp`パッケージによる正規表現マッチに移行。`-i`大文字小文字無視（`(?i)`プレフィックス）、`-v`反転マッチ、`-c`マッチ行数カウント（複数ファイル時はファイル名付き）、`-n`行番号表示、`-r`再帰検索（`filepath.Walk`でディレクトリ展開）。不正な正規表現はexit 2。`grepOptions`構造体でオプション管理。単体テスト22件+統合テスト17件、全38件PASS。

## gf-grep Tier 3: -j JSONフィールド指定検索
- 完了日: 2026-03-14
- 作業内容: `-j` JSONフィールド指定検索オプション追加。`encoding/json`で各行をパースし、指定キーの値に対してのみパターンマッチ。ドット区切りでネストされたキー（例: `user.role`）にも対応。非JSON行はスキップ（マッチなし扱い）。数値・マルチバイト値も`fmt.Sprintf("%v")`で文字列化してマッチ。`-i`/`-v`/`-c`/`-n`など既存オプションとの組み合わせも全て動作。単体テスト12件+統合テスト5件追加、全55件PASS。

## gf-find Tier 1: コア機能
- 完了日: 2026-03-14
- 作業内容: cmd/gf-find/ 作成（go.mod初期化、go.workに追加）。`-name` globパターンによる再帰的ファイル検索、`filepath.Walk`でディレクトリツリー走査、`filepath.Match`でパターンマッチ。パス引数なしの場合はカレントディレクトリ（`.`）をデフォルト使用。複数パス引数対応、単一ファイル引数対応、--version表示、エラーハンドリング（存在しないパス→exit 1）。単体テスト7件+統合テスト11件、全18件PASS。

## gf-find Tier 2: -type f/d、-size、-mtime オプション
- 完了日: 2026-03-14
- 作業内容: `-type f/d`（ファイル/ディレクトリフィルタ）、`-size`（サイズ条件: +N/-N/N + c/k/M/G単位、デフォルト512バイトブロック）、`-mtime`（更新日条件: +N日より前/-N日以内/ちょうどN日前）を追加。`findOptions`構造体でオプション管理、`matchEntry`で全フィルタをAND結合。不正な-type値はexit 2、不正な-size/-mtime式もexit 2。単体テスト25件+統合テスト17件追加、累計60件ALL PASS。

## gf-find Tier 3: -exec 安全版・glob対応
- 完了日: 2026-03-14
- 作業内容: `-exec "command {}"` 確認プロンプト付きコマンド実行オプション追加。`{}`をマッチしたパスに置換し、`sh -c`で実行。実行前に`< command path >?`プロンプトをstderrに表示し、y/yesの場合のみ実行。`-path`フルパスglobマッチオプション追加（`filepath.Match`でパス全体にマッチ）。`-name`との組み合わせもAND結合で動作。`bufio.Reader`でstdin読み取りをキャッシュし複数ファイルの連続プロンプトに対応。単体テスト12件+統合テスト10件追加、累計82件ALL PASS。

## gf-sort Tier 1: コア機能
- 完了日: 2026-03-14
- 作業内容: cmd/gf-sort/ 作成（go.mod初期化、go.workに追加）。辞書順ソート（`sort.Strings`）、stdin対応（引数なし・ハイフン）、複数ファイル入力の結合ソート、--version表示、エラーハンドリング（存在しないファイル→exit 1）。単体テスト4件+統合テスト13件、全17件PASS。

## gf-sort Tier 2: -n 数値ソート、-r 逆順、-k キー指定、-u 重複除去
- 完了日: 2026-03-14
- 作業内容: `-n`数値ソート（`strconv.ParseFloat`）、`-r`逆順ソート、`-k`キーフィールド指定（1-based、`strings.Fields`で分割）、`-u`重複除去（ソート後の隣接重複行を除去）を追加。`sort.SliceStable`で安定ソート。`extractKey`/`parseNumber`/`dedup`関数を実装。全オプションの組み合わせに対応。単体テスト18件+統合テスト16件追加、累計51件ALL PASS。

## gf-sort Tier 3: -t デリミタ指定
- 完了日: 2026-03-14
- 作業内容: `-t`デリミタ指定オプション追加。`extractKey`関数にdelimiterパラメータを追加し、指定時は`strings.Split`でフィールド分割（未指定時は従来通り`strings.Fields`でホワイトスペース分割）。カンマ・コロン・パイプ・タブなど任意のデリミタに対応。`-k`/`-n`/`-r`/`-u`など既存オプションとの組み合わせも全て動作。単体テスト7件+統合テスト7件追加、累計58件ALL PASS。

## gf-uniq Tier 1: コア機能
- 完了日: 2026-03-14
- 作業内容: cmd/gf-uniq/ 作成（go.mod初期化、go.workに追加）。隣接重複行の除去（`processReader`関数で前行と比較）、stdin対応（引数なし・ハイフン）、複数ファイル引数対応、--version表示、エラーハンドリング（存在しないファイル→exit 1）。`run`関数で入出力を分離しテスタビリティ確保。単体テスト14件+統合テスト8件、全22件PASS。

## gf-uniq Tier 2: -c 出現回数カウント、-d 重複行のみ表示、-i 大文字小文字無視
- 完了日: 2026-03-14
- 作業内容: `-c`出現回数カウント（`%7d`フォーマットでプレフィックス表示）、`-d`重複行のみ表示（count<2の行を抑制）、`-i`大文字小文字無視（`strings.EqualFold`で比較）を追加。`uniqOptions`構造体でオプション管理、`compareLine`関数で比較ロジックを分離。`processReader`内で`flush`クロージャにより最終行の出力漏れを防止。全オプション組み合わせ（-c -d、-c -i、-d -i、-c -d -i）に対応。単体テスト14件+run3件+統合テスト9件追加、累計48件ALL PASS。

## gf-uniq Tier 3: --global 非隣接重複除去モード
- 完了日: 2026-03-14
- 作業内容: `--global`フラグ追加。`processReaderGlobal`関数で全行をmapで追跡し、非隣接重複も除去。初出順序をスライスで保持し出現順に出力。`-i`使用時は`strings.ToLower`でキーを正規化。`-c`/-d`/-i`との全組み合わせに対応。単体テスト12件+統合テスト9件追加、累計69件ALL PASS。

## gf-cut Tier 1: コア機能
- 完了日: 2026-03-14
- 作業内容: cmd/gf-cut/ 作成（go.mod初期化、go.workに追加）。`-d`デリミタ指定（デフォルトTAB）＋`-f`フィールド番号指定（単一・複数・範囲・開始範囲・終了範囲対応）、stdin対応（引数なし・ハイフン）、複数ファイル引数対応、--version表示、エラーハンドリング（存在しないファイル→exit 1、-f未指定→exit 2、不正フィールド指定→exit 2）。`parseFields`でフィールド指定パース、`selectFields`でフィールド抽出。単体テスト27件+統合テスト5件、全32件PASS。

## gf-cut Tier 2: -c 文字位置指定・フィールド範囲
- 完了日: 2026-03-14
- 作業内容: `-c`文字位置指定オプション追加。`[]rune`変換によるrune単位の文字位置切り出し。`-f`と`-c`の排他制御（両方指定→exit 2、どちらも未指定→exit 2）。`cutMode`型でフィールドモード/文字モードを切り替え。`selectChars`関数で範囲指定（単一・複数・範囲・開始範囲・終了範囲）に対応。マルチバイト文字（日本語・絵文字）のrune位置も正しく処理。単体テスト8件+統合テスト7件追加、累計47件ALL PASS。

## gf-cut Tier 3: --csv CSV対応モード
- 完了日: 2026-03-14
- 作業内容: `--csv`フラグ追加。`splitCsvFields`関数でダブルクォート内のデリミタを無視するCSVフィールド分割を実装。エスケープされたクォート（`""`)にも対応。クォートは出力にそのまま保持。`-c`との排他制御（`--csv`+`-c`→exit 2）。マルチバイト文字・タブデリミタ・複数クォートフィールドに対応。単体テスト9件+統合テスト9件追加、累計65件ALL PASS。

## gf-sed Tier 1: コア機能
- 完了日: 2026-03-14
- 作業内容: cmd/gf-sed/ 作成（go.mod初期化、go.workに追加）。`s/pattern/replace/` 基本置換（1行に最初の1つ）、正規表現対応、カスタムデリミタ（`s|pat|rep|`）、エスケープされたデリミタ、キャプチャグループ置換、stdin対応（引数なし・ハイフン）、複数ファイル対応、--version表示、エラーハンドリング（存在しないファイル→exit 1、不正式→exit 2）。単体テスト25件+統合テスト10件+その他1件、全36件PASS。

## gf-sed Tier 2: g フラグ・アドレス指定・-i in-place編集
- 完了日: 2026-03-14
- 作業内容: `g`フラグ（全置換、`ReplaceAllString`使用）、アドレス指定（行番号`Ns/...`、最終行`$s/...`、パターン`/pat/s/...`）、`-i` in-place編集（ファイル読み込み→変換→同一パスに書き戻し、パーミッション保持）を追加。`parseAddress`関数でアドレスプレフィックスをパース、`matchAddress`で行ごとの適用判定。`$`アドレスは全行読み込み後に最終行のみ変換する`processReaderLastLine`で処理。`runInPlace`関数でin-place編集を実装。単体テスト33件追加、累計62件ALL PASS。

## gf-sed Tier 3: マルチバイト安全な置換（rune単位処理）
- 完了日: 2026-03-14
- 作業内容: 式パーサーをrune単位処理に全面改修。`splitByDelim`を`byte`から`rune`ベースに変更し、マルチバイトデリミタ（★、🔥等）に対応。`parseExpression`で`utf8.DecodeRuneInString`によるデリミタ抽出。`findClosingSlash`もrune単位走査に変更。`parseAddress`の先頭文字判定もrune安全に。CJK文字・絵文字（4バイト）・全角数字・結合文字・ゼロ幅文字のパターンマッチ＋置換テスト、マルチバイトデリミタテスト、in-placeマルチバイトテストを追加。単体テスト19件追加、累計81件ALL PASS。

## gf-xargs Tier 1: コア機能
- 完了日: 2026-03-14
- 作業内容: cmd/gf-xargs/ 作成（go.mod初期化、go.workに追加）。stdinから行を読み取り、ホワイトスペースでトークン分割し、指定コマンドの引数として実行。コマンド未指定時はecho。シングル/ダブルクォート対応（クォート内スペース保持）。--version表示、エラーハンドリング（コマンド失敗→exit 1、不正フラグ→exit 2、空stdin→実行なし）。commandExecutorインターフェースでテスタビリティ確保。単体テスト17件+エッジケース4件+統合テスト9件、全30件PASS。

## gf-xargs Tier 2: -n 最大引数数指定・-P 並列実行数指定
- 完了日: 2026-03-14
- 作業内容: `-n`最大引数数指定（`splitBatches`関数でアイテムをN個ずつバッチ分割、各バッチごとにコマンド実行）、`-P`並列実行数指定（`runParallel`関数でセマフォ＋goroutineによる並列実行、出力バッファリングでインターリーブ防止、`sync.Mutex`でスレッドセーフな出力）。`-n`負値→exit 2、`-P`が0以下→exit 2のバリデーション。単体テスト16件（splitBatches 6件、runWithN 5件、runWithP 5件）+統合テスト3件追加、累計48件ALL PASS。

## gf-xargs Tier 3: -0 null区切り対応・--dry-run コマンド表示
- 完了日: 2026-03-14
- 作業内容: `-0`null区切り対応（`readItemsNull`関数でnullバイト区切りのカスタムスキャナー、スペース・改行を含むアイテムをそのまま保持）、`--dry-run`コマンド表示（`shellJoin`/`shellQuote`で安全なシェルコマンド文字列を生成、実行せずstdoutに出力）。`-0`と`-n`/`-P`の組み合わせ、`--dry-run`と`-n`/`-0`の組み合わせも全て動作。単体テスト21件（readItemsNull 8件、shellJoin 5件、runWithNullDelim 3件、runDryRun 5件）+統合テスト4件追加、累計69件ALL PASS。

## gf-diff Tier 1: コア機能
- 完了日: 2026-03-14
- 作業内容: cmd/gf-diff/ 作成（go.mod初期化、go.workに追加）。2ファイルの行単位diff（Myers algorithm）。`myersDiff`関数でforward pass+backtrackによる最短編集スクリプト算出。出力は`< `（削除）、`> `（挿入）、`  `（同一）形式。差分あり→exit 1、差分なし→exit 0。--version表示、エラーハンドリング（ファイル未検出→exit 1、引数不正→exit 2）。マルチバイト対応、1000行大量入力テスト。単体テスト12件+統合テスト7件+エラー系6件+大量入力1件+バージョン1件、全28件PASS。

## gf-diff Tier 2: unified diff format (`-u`) 出力
- 完了日: 2026-03-14
- 作業内容: `-u`フラグでunified diff format出力を追加。`buildHunks`関数でeditsリストからコンテキスト付きhunkを生成（デフォルト3行コンテキスト）。`printUnified`関数で`--- file1`/`+++ file2`ヘッダ＋`@@ -x,y +a,b @@` hunkヘッダ＋`-`/`+`/` `行を出力。近接した変更はhunkをマージ、離れた変更は分割。空ファイル・マルチバイト対応。単体テスト6件(buildHunks)+統合テスト8件(unified format)追加、累計42件ALL PASS。

## gf-diff Tier 3: カラー出力・`--word` 単語単位diff
- 完了日: 2026-03-14
- 作業内容: `--color=auto|always|never`フラグ追加。`isTerminal`関数でターミナル検出（`os.ModeCharDevice`判定）、autoモードは非ターミナル時カラー無効。削除行を赤、挿入行を緑、unifiedヘッダ（---を太字赤、+++を太字緑、@@をシアン）でカラー出力。`--word`フラグ追加。`splitWords`関数でrune単位の単語/空白トークン分割。`wordDiffLine`関数で隣接delete/insertペアの行内単語をMyersアルゴリズムで比較し、`[-old-]`/`[+new+]`マーカーで差分表示。通常モード・unifiedモード両対応。`--color`と`--word`の併用も動作。マルチバイト対応。単体テスト14件(splitWords 8件+wordDiffLine 6件)+統合テスト12件(color 5件+word diff 6件+大量入力1件)追加、累計76件ALL PASS。

## gf-tree Tier 1: コア機能
- 完了日: 2026-03-14
- 作業内容: cmd/gf-tree/ 作成（go.mod初期化、go.workに追加）。再帰的ディレクトリツリー描画（罫線文字 ├──、└──、│ 使用）。`os.ReadDir`でエントリ取得、アルファベット順ソート。ルートディレクトリ名表示、サマリー行（N directories, N files）表示。`treeStats`構造体でディレクトリ・ファイル数を再帰的に集計。stdin対応なし（ディレクトリパス引数必須）、引数なし時はカレントディレクトリ、複数ディレクトリ対応。--version表示、エラーハンドリング（存在しないパス→exit 1、ファイルパス→エラー）。マルチバイトファイル名対応。単体テスト10件（walkDir 7件+printTree 3件）+統合テスト8件、全18件PASS。

## gf-tree Tier 2: -L 深さ制限・-I 除外パターン
- 完了日: 2026-03-14
- 作業内容: `-L`深さ制限オプション追加（0=無制限、1以上で指定階層まで表示、負値→exit 2）。`-I`除外パターンオプション追加（`filepath.Match`によるglob照合、全階層で適用）。`treeOptions`構造体でオプション管理、`walkDir`に`depth`パラメータ追加。`isExcluded`関数でエントリ名のglob判定。フィルタ後のエントリリストで末尾コネクタ（└──）を正しく決定。`-L`と`-I`の組み合わせも動作。単体テスト14件（深さ制限4件+除外パターン5件+isExcluded 5件）+統合テスト6件追加、累計38件ALL PASS。

## gf-tree Tier 3: ファイルサイズ表示・--du ディレクトリサイズ集計
- 完了日: 2026-03-14
- 作業内容: `-s`ファイルサイズ表示オプション追加（`[サイズ]  ファイル名`形式）。`--du`ディレクトリサイズ集計オプション追加（`calcDirSize`関数で再帰的にサイズ合算、ディレクトリ・ファイル両方にサイズ表示）。`formatSize`関数で人間可読形式（B/K/M/G）に変換。`--du`はルートディレクトリにも合計サイズを表示。`-L`深さ制限時も`calcDirSize`で制限なく全サイズ集計。`-I`除外パターンはサイズ集計にも反映。単体テスト13件（formatSize 8件+サイズ表示2件+duディレクトリサイズ2件+calcDirSize 1件）+統合テスト5件追加、累計57件ALL PASS。

## gf-jq Tier 1: 基本パスアクセス・stdin対応
- 完了日: 2026-03-14
- 作業内容: cmd/gf-jq/ 作成（go.mod初期化、go.workに追加）。`.key`、`.key.nested`、`.[0]`の基本パスアクセス。`parseFilter`関数でフィルタ式をトークン列にパース、`applyFilter`関数でJSON値にフィルタ適用。負のインデックス対応、存在しないキー/範囲外インデックスはnull返却。`encoding/json`でパース、型別出力（文字列はクォート付き、整数は小数点なし、オブジェクト/配列はインデント付き）。stdin対応（引数なし・ハイフン）、複数ファイル対応、--version表示、エラーハンドリング。マルチバイトキー対応。単体テスト12件（parseFilter）+20件（applyFilter）+2件（errors）+2件（invalidJSON/empty）+統合テスト9件、全45件PASS。

## gf-jq Tier 2: パイプ・配列操作・length
- 完了日: 2026-03-14
- 作業内容: フィルタシステムをパイプライン方式（`[][]token`）に全面リファクタ。`|`でフィルタを複数ステージに分割し、各ステージの出力を次のステージの入力に渡す`applyPipeline`/`applyStage`関数を実装。`.[]`配列/オブジェクトイテレータ追加（配列は全要素展開、オブジェクトはキーのアルファベット順で値を展開、ファンアウト対応）。`length`組み込み関数追加（配列→要素数、オブジェクト→キー数、文字列→rune数、null→0、数値→絶対値）。`.items[]`や`.[].name`のような単一ステージ内でのイテレータ+アクセスの組み合わせ、`.[] | .items | length`のような複数ステージパイプラインも全て動作。テスト30件追加（Iterator 12件、IteratorErrors 4件、Length 13件、LengthErrors 1件、Pipe 5件、Run追加3件、UnknownFunction 1件、ParseFilter追加14件）、累計75件ALL PASS。

## gf-jq Tier 3: select(条件)フィルタ・keys・values
- 完了日: 2026-03-14
- 作業内容: `keys`組み込み関数追加（オブジェクト→ソート済みキー配列、配列→インデックス配列）。`values`組み込み関数追加（オブジェクト→キーソート順の値配列、配列→そのまま）。`select(条件)`フィルタ追加。比較演算子（`==`/`!=`/`>`/`<`/`>=`/`<=`）による数値・文字列・null比較、演算子なしのtruthiness判定（null/falseを除外）に対応。`parseSelectCondition`で条件式パース、`evalSelect`で条件評価、`compareValues`で型別比較。`splitPipeline`関数で括弧内の`|`を保護するパイプ分割に改良。ネストされたキーアクセス・マルチバイト文字列・selectの後続パイプラインとの組み合わせも全て動作。テスト39件追加（Keys 6件、KeysErrors 3件、Values 4件、ValuesErrors 2件、Select 15件、SelectErrors 2件、ParseFilterTier3 7件）、累計114件ALL PASS。

## gf-hexdump Tier 1: 16進ダンプ表示・stdin対応
- 完了日: 2026-03-14
- 作業内容: cmd/gf-hexdump/ 作成（go.mod初期化、go.workに追加）。16バイトずつの16進ダンプ表示（`%08x`オフセット＋hex bytes＋ASCII表示）。`io.ReadFull`で16バイトずつ読み取り、`formatLine`関数でオフセット・16進バイト（8バイトずつグループ化）・ASCII表示（非印字文字は`.`）を1行に出力。stdin対応（引数なし・ハイフン）、複数ファイル対応、--version表示、エラーハンドリング（存在しないファイル→exit 1、不正フラグ→exit 2）。単体テスト5件（formatLine）+7件（hexdump）+9件（run）+2件（境界テスト）、全23件PASS。

## gf-hexdump Tier 2: -s オフセット指定・-n バイト数制限
- 完了日: 2026-03-14
- 作業内容: `-s`オフセット指定オプション追加（`io.Seeker`対応時はSeek、非対応時は`io.CopyN`でスキップ）。`-n`バイト数制限オプション追加（`io.LimitReader`で読み取りバイト数を制限）。`hexdumpOptions`構造体でオプション管理。skip後のオフセット表示は実際のファイル位置を反映。負値バリデーション（-s負値→exit 2、-n負値→exit 2）。ファイル・stdinの両方で動作。単体テスト10件（skip 4件+limit 4件+組み合わせ2件）+統合テスト7件追加、累計40件ALL PASS。

## gf-hexdump Tier 3: カラー出力（NULL, 印字可能, 制御文字で色分け）
- 完了日: 2026-03-14
- 作業内容: `--color`フラグ追加（auto/always/never）。`byteColor`関数でバイト値に応じた色分け（NULL→dim、印字可能0x20-0x7e→緑、制御文字0x01-0x1f,0x7f→赤、高バイト0x80-0xff→青）。hexバイト部分とASCII部分の両方をカラー化。autoモードは`os.ModeCharDevice`でターミナル検出。不正なカラーモードはexit 2。単体テスト22件追加（byteColor 11件+formatLineColor 5件+colorNoColor 1件+runColorFlag 4件+既存1件）、累計62件ALL PASS。

## gf-claude-quota Phase 1: MVP — 型定義・APIクライアント・Keychainトークン取得・基本CLI
- 完了日: 2026-03-14
- 作業内容: `cmd/gf-claude-quota/` 作成（go.mod初期化、go.workに追加）。`internal/api/types.go`（UsageResponse, UsageWindow構造体）。`internal/api/client.go`（net/httpでGET、JSONパース、Authorization/anthropic-betaヘッダー設定、401/429/5xxエラーハンドリング、SetEndpointでテスト用エンドポイント差し替え）。`internal/credentials/keychain.go`（macOS `security find-generic-password`コマンドでKeychain読み取り、JSONパースしてaccessToken抽出、CommandRunnerインターフェースでテスタビリティ確保）。`main.go`（トークン取得→API呼び出し→テキスト形式でプログレスバー付き使用率表示、buildBar/formatResetTime/printWindow関数）。モックHTTPサーバーでAPIクライアントのテスト（正常系・エラー系・不正JSON・全null・高使用率）、Keychainパースロジックのテスト（正常パース・コマンド失敗・不正JSON・空白トリミング）、main関数テスト（--version・不正フラグ・buildBar・formatResetTime）。テスト14件（サブテスト30件以上）、全PASS。

## gf-claude-quota Phase 2: ファイルキャッシュ
- 完了日: 2026-03-14
- 作業内容: `internal/cache/filecache.go` — FileCache構造体（Get/Set/isStale）、`~/.cache/gf-claude-quota/usage.json`にAPIレスポンスをJSON保存、TTLベース有効期限管理、キャッシュディレクトリ自動作成、アトミック書き込み（tmp+rename）、ファイルパーミッション0600。main.goにキャッシュチェック→API呼び出しのフローを統合。`--cache-ttl`（デフォルト60秒）、`--no-cache`フラグ追加。キャッシュファイルにトークンが含まれないことを検証。テスト13件追加、累計27件ALL PASS。

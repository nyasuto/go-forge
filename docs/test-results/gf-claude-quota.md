# gf-claude-quota テスト結果

## Phase 1: MVP — 型定義・APIクライアント・Keychainトークン取得・基本CLI

### 実行日: 2026-03-14

### テスト結果: ALL PASS

### テスト内訳

#### main_test.go (4件)
- TestBuildBar: プログレスバー生成（0%, 50%, 100%, 42%, 99.5%, 負値, 100超）— 7サブテスト
- TestFormatResetTime: リセット時刻フォーマット（時分, 日時, 分のみ, 過去, 不正形式）— 5サブテスト
- TestFormatUsage: 使用量表示のフォーマット確認
- TestRun_Version: --version フラグ
- TestRun_InvalidFlag: 不正フラグでexit 2

#### internal/api/client_test.go (5件)
- TestFetchUsage_Success: 正常レスポンスのパース、ヘッダー検証
- TestFetchUsage_Errors: HTTPエラーハンドリング（401, 429, 500, 503, 403）— 5サブテスト
- TestFetchUsage_InvalidJSON: 不正JSONレスポンス
- TestFetchUsage_AllFieldsNull: 全フィールドnull
- TestFetchUsage_HighUtilization: 高使用率（99.5%, 100%）
- TestNewClient_NilHTTPClient: nilクライアントでデフォルト使用

#### internal/credentials/keychain_test.go (4件)
- TestParseKeychainJSON: Keychain JSONパース（正常, Max, 不正JSON, missing fields, null, empty）— 7サブテスト
- TestGetTokenFromKeychain_Success: モックCommandRunnerでトークン取得
- TestGetTokenFromKeychain_CommandFailure: コマンド失敗時エラー
- TestGetTokenFromKeychain_InvalidJSON: 不正JSON応答
- TestGetTokenFromKeychain_WhitespaceHandling: 空白トリミング

### テスト合計: 14件（サブテスト含む30件以上）、全PASS

---

## Phase 2: ファイルキャッシュ

### 実行日: 2026-03-14

### テスト結果: ALL PASS

### テスト内訳

#### internal/cache/filecache_test.go (13件)
- TestFileCache_SetAndGet: Set→Getで正しいデータ取得
- TestFileCache_GetStale: TTL超過でキャッシュミス（nil返却）
- TestFileCache_GetNotStale: TTL内でキャッシュヒット
- TestFileCache_GetMissing: ファイル未存在でnil返却（エラーなし）
- TestFileCache_GetCorruptJSON: 不正JSONはキャッシュミス扱い
- TestFileCache_SetCreatesDirectory: ネストされたディレクトリ自動作成
- TestFileCache_SetOverwrite: 上書き更新
- TestFileCache_FilePermissions: ファイルパーミッション0600
- TestFileCache_CacheFileContents: キャッシュファイルの中身検証（fetchedAt, usage）
- TestFileCache_DefaultDir: デフォルトディレクトリ（~/.cache/gf-claude-quota/）
- TestFileCache_NullFields: 全フィールドnilのUsageResponseキャッシュ
- TestFileCache_CustomTTL: カスタムTTL（5秒）の動作確認
- TestFileCache_NoTokenInCache: キャッシュファイルにトークンが含まれないことを確認

#### main.go 変更
- `--cache-ttl` フラグ追加（デフォルト60秒）
- `--no-cache` フラグ追加
- キャッシュチェック→API呼び出しのフロー統合

### テスト合計: 27件（Phase 1: 14件 + Phase 2: 13件）、全PASS

---

## Phase 3: 出力フォーマッタ（JSON・oneline・プログレスバー・カラー）

### 実行日: 2026-03-14

### テスト結果: ALL PASS

### テスト内訳

#### internal/output/output_test.go (19件)
- TestBuildBar: プログレスバー生成（0%, 50%, 100%, 42%, 99.5%, 負値, 100超, width20, 1%, 5%）— 10サブテスト
- TestFormatResetTime: リセット時刻フォーマット（時分, 日時, 分のみ, 過去, 不正形式）— 5サブテスト
- TestColorLevel: カラー閾値判定（0%緑, 49%緑, 50%黄, 79%黄, 80%赤, 100%赤）— 6サブテスト
- TestColorize: ANSIカラーコード付与（緑, 黄, 赤）— 3サブテスト
- TestParseColorMode: カラーモードパース（auto, always, never, invalid, 空文字）— 5サブテスト
- TestFormatText: テキストモード出力（ヘッダ, ラベル, 使用率, リセット時間）
- TestFormatText_WithColor: カラー付きテキスト出力（ANSIコード存在確認）
- TestFormatText_NoColor: カラーなしテキスト出力（ANSIコード不在確認）
- TestFormatText_NilWindows: 全windowがnilの場合のヘッダのみ出力
- TestFormatText_HighUtilization: 高使用率（95%）で赤色確認
- TestFormatText_MediumUtilization: 中使用率（65%）で黄色確認
- TestFormatJSON: JSON出力（全フィールド, resets_in, 構造検証）
- TestFormatJSON_NilWindows: nilフィールドのomitempty確認
- TestFormatJSON_ValidJSON: インデント付きJSON検証
- TestFormatOneline: ワンライナー出力（5h:42%(2h29m) 7d:18%形式）
- TestFormatOneline_WithOpus: Opus使用率>0の場合の出力確認
- TestFormatOneline_NilWindows: 全windowがnilの場合の空出力
- TestFormatOneline_NoResetTime: リセット時刻なしの場合

#### main_test.go (4件)
- TestRun_Version: --version フラグ
- TestRun_InvalidFlag: 不正フラグでexit 2
- TestRun_InvalidColorMode: 不正カラーモードでexit 2
- TestRun_MutuallyExclusiveFlags: --json + --oneline の排他制御でexit 2

#### 新規ファイル
- `internal/output/bar.go` — BuildBar, FormatResetTime, ColorLevel, Colorize関数
- `internal/output/text.go` — FormatText（カラー対応テキスト出力）, ParseColorMode, ShouldColorize
- `internal/output/json.go` — FormatJSON（インデント付きJSON）, FormatOneline（コンパクト1行形式）

#### main.go 変更
- `--json` フラグ追加（JSON形式出力）
- `--oneline` フラグ追加（ワンライナー出力）
- `--color` フラグ追加（auto|always|never）
- 出力ロジックを internal/output パッケージに分離
- printUsage関数でモード別出力を統合

### テスト合計: 50件（Phase 1: 14件→4件に整理 + Phase 2: 13件 + Phase 3: 23件 + api: 6件 + credentials: 5件）、全PASS

---

## Phase 4: statusLine統合・formatテンプレート

### 実行日: 2026-03-14

### テスト結果: ALL PASS

### テスト内訳

#### internal/output/output_test.go (23件追加)
- TestFormatStatusLine_WithStdinData: stdinからJSON読み取り、quota+model+ctx+cost合成出力
- TestFormatStatusLine_NoStdinData: stdin無しでquotaのみ表示にフォールバック
- TestFormatStatusLine_EmptyStdin: 空stdin対応
- TestFormatStatusLine_InvalidStdinJSON: 不正JSONでもquota出力継続
- TestFormatStatusLine_NilWindows: quotaなしでstdinデータのみ表示
- TestFormatStatusLine_WithOpus: Opus使用率>0の場合の出力確認
- TestFormatStatusLine_ZeroCostNotShown: cost=0の場合は$0を表示しない
- TestFormatTemplate_BasicVars: 基本変数置換（{5h}, {7d}）
- TestFormatTemplate_WithStdinVars: stdin変数（{model}, {ctx_pct}, {cost}）
- TestFormatTemplate_ResetTimeVar: リセット時刻変数（{5h_reset}）
- TestFormatTemplate_BarVar: プログレスバー変数（{5h_bar}）
- TestFormatTemplate_OpusVar: Opus変数（{opus}）
- TestFormatTemplate_NilWindows: 全windowがnilの場合N/A表示
- TestFormatTemplate_NoStdinVars: stdin無しで空文字変数
- TestBuildTemplateVars: テンプレート変数構築の全項目検証
- TestBuildTemplateVars_NilInput: nil入力時のデフォルト値検証
- TestStatusLineInput_Parsing: JSON入力パース（full, partial, empty）— 3サブテスト

#### main_test.go (5件に拡張)
- TestRun_MutuallyExclusiveFlags: 排他制御を5パターンに拡張（json+oneline, json+statusline, oneline+statusline, json+format, statusline+format）

#### 新規ファイル
- `internal/output/statusline.go` — FormatStatusLine（statusLine合成出力）, FormatTemplate（テンプレートエンジン）, StatusLineInput構造体, buildQuotaParts, buildTemplateVars

#### main.go 変更
- `--statusline` フラグ追加（statusLine形式出力）
- `--format` フラグ追加（カスタムテンプレート出力）
- stdin読み取り対応（`io.Reader`パラメータ追加）
- 排他制御を4モード対応に拡張

### テスト合計: 73件（main: 9件 + api: 6件 + cache: 13件 + credentials: 5件 + output: 40件）、全PASS

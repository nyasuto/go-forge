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

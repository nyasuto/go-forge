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

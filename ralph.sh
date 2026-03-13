#!/bin/bash

# 無限ループやAPI課金の暴走を防ぐための上限回数
MAX_LOOPS=10

for ((i=1; i<=MAX_LOOPS; i++)); do
    echo "========================================"
    echo " Ralph Loop $i 回目を開始します..."
    echo "========================================"
    
    # Claude Codeにプロンプトファイルを渡して実行
    # (-p オプションで初期プロンプトを渡します)
    claude --dangerously-skip-permissions -p "$(cat prompt.md)"
    
    echo "========================================"
    echo " Loop $i 終了（コンテキストをリセット）"
    echo " 次のループまで5秒待機します..."
    echo "========================================"
    sleep 5
done

echo "指定された最大ループ回数 ($MAX_LOOPS 回) に到達しました。"
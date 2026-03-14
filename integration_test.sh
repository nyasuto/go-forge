#!/bin/bash
# GoForge Integration Test — パイプチェーン連携テスト
# 全ツールのパイプライン連携を検証する

set -euo pipefail

PASS=0
FAIL=0
ERRORS=""

# ツールバイナリのパス
BIN="$(cd "$(dirname "$0")" && pwd)"

# 各ツールのバイナリパスを設定
GF_CAT="$BIN/cmd/gf-cat/gf-cat"
GF_HEAD="$BIN/cmd/gf-head/gf-head"
GF_TAIL="$BIN/cmd/gf-tail/gf-tail"
GF_WC="$BIN/cmd/gf-wc/gf-wc"
GF_TEE="$BIN/cmd/gf-tee/gf-tee"
GF_GREP="$BIN/cmd/gf-grep/gf-grep"
GF_FIND="$BIN/cmd/gf-find/gf-find"
GF_SORT="$BIN/cmd/gf-sort/gf-sort"
GF_UNIQ="$BIN/cmd/gf-uniq/gf-uniq"
GF_CUT="$BIN/cmd/gf-cut/gf-cut"
GF_SED="$BIN/cmd/gf-sed/gf-sed"
GF_XARGS="$BIN/cmd/gf-xargs/gf-xargs"
GF_DIFF="$BIN/cmd/gf-diff/gf-diff"
GF_TREE="$BIN/cmd/gf-tree/gf-tree"
GF_JQ="$BIN/cmd/gf-jq/gf-jq"
GF_HEXDUMP="$BIN/cmd/gf-hexdump/gf-hexdump"

# テスト用一時ディレクトリ
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

# テストヘルパー
assert_eq() {
    local name="$1"
    local expected="$2"
    local actual="$3"
    if [ "$expected" = "$actual" ]; then
        PASS=$((PASS + 1))
        echo "  PASS: $name"
    else
        FAIL=$((FAIL + 1))
        ERRORS="$ERRORS\n  FAIL: $name\n    expected: $(echo "$expected" | head -3)\n    actual:   $(echo "$actual" | head -3)"
        echo "  FAIL: $name"
        echo "    expected: $(echo "$expected" | head -3)"
        echo "    actual:   $(echo "$actual" | head -3)"
    fi
}

# テストデータ作成
cat > "$TMPDIR/fruits.txt" << 'EOF'
apple
banana
cherry
apple
banana
fig
grape
apple
EOF

cat > "$TMPDIR/nums.txt" << 'EOF'
3
1
4
1
5
9
2
6
5
3
5
EOF

cat > "$TMPDIR/data.csv" << 'EOF'
name,age,city
Alice,30,Tokyo
Bob,25,Osaka
Charlie,35,Tokyo
Diana,28,Nagoya
Eve,30,Tokyo
EOF

cat > "$TMPDIR/sample.go" << 'EOF'
package main

import "fmt"

func hello() {
    fmt.Println("Hello")
}

func world() {
    fmt.Println("World")
}

func main() {
    hello()
    world()
}
EOF

cat > "$TMPDIR/data.json" << 'EOF'
[
  {"name": "Alice", "age": 30, "city": "Tokyo"},
  {"name": "Bob", "age": 25, "city": "Osaka"},
  {"name": "Charlie", "age": 35, "city": "Tokyo"},
  {"name": "Diana", "age": 28, "city": "Nagoya"}
]
EOF

cat > "$TMPDIR/log.txt" << 'EOF'
2024-01-01 INFO  Server started
2024-01-01 ERROR Connection failed
2024-01-02 INFO  Request received
2024-01-02 WARN  High latency detected
2024-01-03 ERROR Database timeout
2024-01-03 INFO  Recovery complete
EOF

# サブディレクトリ作成
mkdir -p "$TMPDIR/src/pkg"
echo 'func Foo() {}' > "$TMPDIR/src/foo.go"
echo 'func Bar() {}' > "$TMPDIR/src/bar.go"
echo 'func Baz() {}' > "$TMPDIR/src/pkg/baz.go"
echo 'not a go file' > "$TMPDIR/src/readme.txt"

echo "=== GoForge Integration Tests ==="
echo ""

# ───────────────────────────────────
echo "--- Test 1: gf-find | gf-grep | gf-wc ---"
# .goファイルを探して、funcを含む行を数える
result=$("$GF_FIND" -name "*.go" "$TMPDIR/src" | "$GF_XARGS" "$GF_GREP" "func" | "$GF_WC" -l)
result=$(echo "$result" | tr -d ' ')
assert_eq "find .go files | grep func | wc -l" "3" "$result"

# ───────────────────────────────────
echo "--- Test 2: gf-cat | gf-sort | gf-uniq ---"
# ファイルを読み込み、ソートして重複除去
result=$("$GF_CAT" "$TMPDIR/fruits.txt" | "$GF_SORT" | "$GF_UNIQ")
expected="apple
banana
cherry
fig
grape"
assert_eq "cat | sort | uniq" "$expected" "$result"

# ───────────────────────────────────
echo "--- Test 3: gf-cat | gf-sort | gf-uniq -c | gf-sort -n -r ---"
# 出現回数カウント → 数値逆順ソート（頻度ランキング）
result=$("$GF_CAT" "$TMPDIR/fruits.txt" | "$GF_SORT" | "$GF_UNIQ" -c | "$GF_SORT" -n -r -k 1 | "$GF_HEAD" -n 3)
# uniq -c は "%7d " 形式
line1=$(echo "$result" | head -1 | tr -s ' ' | sed 's/^ //')
assert_eq "frequency ranking top" "3 apple" "$line1"

# ───────────────────────────────────
echo "--- Test 4: gf-cat | gf-grep | gf-wc ---"
# ログからERROR行を抽出して行数を数える
result=$("$GF_CAT" "$TMPDIR/log.txt" | "$GF_GREP" "ERROR" | "$GF_WC" -l)
result=$(echo "$result" | tr -d ' ')
assert_eq "cat log | grep ERROR | wc -l" "2" "$result"

# ───────────────────────────────────
echo "--- Test 5: gf-cat | gf-cut | gf-sort | gf-uniq ---"
# CSVから都市列を抽出 → ソート → 重複除去
result=$("$GF_CAT" "$TMPDIR/data.csv" | "$GF_TAIL" -n 5 | "$GF_CUT" -d ',' -f 3 | "$GF_SORT" | "$GF_UNIQ")
expected="Nagoya
Osaka
Tokyo"
assert_eq "csv city column | sort | uniq" "$expected" "$result"

# ───────────────────────────────────
echo "--- Test 6: gf-cat | gf-sed | gf-grep ---"
# sedで置換してからgrepで検索
result=$("$GF_CAT" "$TMPDIR/log.txt" | "$GF_SED" 's/ERROR/CRITICAL/g' | "$GF_GREP" "CRITICAL" | "$GF_WC" -l)
result=$(echo "$result" | tr -d ' ')
assert_eq "cat | sed s/ERROR/CRITICAL/ | grep CRITICAL | wc" "2" "$result"

# ───────────────────────────────────
echo "--- Test 7: gf-cat | gf-head | gf-tail ---"
# 3行目だけを取得（head -n 3 | tail -n 1）
result=$("$GF_CAT" "$TMPDIR/fruits.txt" | "$GF_HEAD" -n 3 | "$GF_TAIL" -n 1)
assert_eq "head -n 3 | tail -n 1 (line 3)" "cherry" "$result"

# ───────────────────────────────────
echo "--- Test 8: gf-sort -n | gf-head | gf-tail ---"
# 数値ソート → 中央値付近の取得
result=$("$GF_CAT" "$TMPDIR/nums.txt" | "$GF_SORT" -n -u | "$GF_HEAD" -n 5 | "$GF_TAIL" -n 1)
assert_eq "sort -n -u | head -n 5 | tail -n 1 (median area)" "5" "$result"

# ───────────────────────────────────
echo "--- Test 9: gf-cat | gf-tee | gf-wc ---"
# teeで分岐しながらwcでカウント
tee_out="$TMPDIR/tee_output.txt"
result=$("$GF_CAT" "$TMPDIR/fruits.txt" | "$GF_TEE" "$tee_out" | "$GF_WC" -l)
result=$(echo "$result" | tr -d ' ')
assert_eq "cat | tee file | wc -l (stdout)" "8" "$result"
# teeで書き出されたファイルの行数も確認
tee_wc=$("$GF_WC" -l "$tee_out")
tee_wc=$(echo "$tee_wc" | awk '{print $1}')
assert_eq "tee output file line count" "8" "$tee_wc"

# ───────────────────────────────────
echo "--- Test 10: gf-jq | gf-grep ---"
# JSONから名前を展開してgrepでフィルタ
result=$("$GF_CAT" "$TMPDIR/data.json" | "$GF_JQ" '.[] | .name' | "$GF_GREP" -i "^\"[ab]")
expected='"Alice"
"Bob"'
assert_eq "jq .[] | .name | grep A or B" "$expected" "$result"

# ───────────────────────────────────
echo "--- Test 11: gf-cat | gf-grep | gf-cut ---"
# CSVから名前列を抽出してgrepでフィルタ
result=$("$GF_CAT" "$TMPDIR/data.csv" | "$GF_TAIL" -n 5 | "$GF_CUT" -d ',' -f 1 | "$GF_GREP" -i "^[a-c]")
expected="Alice
Bob
Charlie"
assert_eq "csv names | grep A-C" "$expected" "$result"

# ───────────────────────────────────
echo "--- Test 12: gf-find | gf-sort | gf-head ---"
# ファイル検索結果をソートして先頭N件
result=$("$GF_FIND" -name "*.go" -type f "$TMPDIR/src" | "$GF_SORT" | "$GF_HEAD" -n 2)
lines=$(echo "$result" | wc -l | tr -d ' ')
assert_eq "find *.go | sort | head -n 2 (line count)" "2" "$lines"

# ───────────────────────────────────
echo "--- Test 13: gf-cat | gf-sed | gf-sort | gf-uniq -c ---"
# 全て小文字に変換 → ソート → カウント
# sed \L& はGoのsedでは非対応のため、gf-sed で a-z 変換テスト
result=$(echo -e "HELLO\nworld\nHELLO\nWorld" | "$GF_SED" 's/HELLO/hello/g' | "$GF_SORT" | "$GF_UNIQ" -c | "$GF_SORT" -n -r -k 1 | "$GF_HEAD" -n 1)
result=$(echo "$result" | tr -s ' ' | sed 's/^ //')
assert_eq "sed replace | sort | uniq -c | sort -nr | head" "2 hello" "$result"

# ───────────────────────────────────
echo "--- Test 14: gf-hexdump | gf-head ---"
# hexdumpの出力を先頭2行だけ取得
result=$(echo -n "Hello, World!" | "$GF_HEXDUMP" | "$GF_HEAD" -n 1)
# 最初の行にオフセット00000000が含まれること
echo "$result" | grep -q "00000000" && {
    PASS=$((PASS + 1))
    echo "  PASS: hexdump | head -n 1 contains offset"
} || {
    FAIL=$((FAIL + 1))
    echo "  FAIL: hexdump | head -n 1 contains offset"
    ERRORS="$ERRORS\n  FAIL: hexdump | head missing offset"
}

# ───────────────────────────────────
echo "--- Test 15: gf-jq | gf-wc ---"
# JSON配列の要素数を確認
result=$("$GF_CAT" "$TMPDIR/data.json" | "$GF_JQ" '.[] | .city' | "$GF_SORT" | "$GF_UNIQ" -c | "$GF_SORT" -n -r -k 1 | "$GF_HEAD" -n 1)
result=$(echo "$result" | tr -s ' ' | sed 's/^ //')
assert_eq "jq cities | sort | uniq -c | sort -nr top" "2 \"Tokyo\"" "$result"

# ───────────────────────────────────
echo "--- Test 16: gf-cat | gf-grep -v | gf-sed | gf-wc ---"
# grep -v で除外 → sed で変換 → wc
result=$("$GF_CAT" "$TMPDIR/log.txt" | "$GF_GREP" -v "INFO" | "$GF_SED" 's/ERROR/ERR/g' | "$GF_WC" -l)
result=$(echo "$result" | tr -d ' ')
assert_eq "grep -v INFO | sed | wc -l" "3" "$result"

# ───────────────────────────────────
echo "--- Test 17: multi-file gf-cat | gf-sort | gf-uniq | gf-wc ---"
# 複数ファイルをcatで結合 → ソート → 重複除去 → 行数カウント
result=$("$GF_CAT" "$TMPDIR/fruits.txt" "$TMPDIR/fruits.txt" | "$GF_SORT" | "$GF_UNIQ" | "$GF_WC" -l)
result=$(echo "$result" | tr -d ' ')
assert_eq "cat file file | sort | uniq | wc -l" "5" "$result"

# ───────────────────────────────────
echo "--- Test 18: gf-diff pipe ---"
# 2つのファイルを作ってdiffの出力をgrepでフィルタ
echo -e "line1\nline2\nline3" > "$TMPDIR/a.txt"
echo -e "line1\nmodified\nline3" > "$TMPDIR/b.txt"
result=$("$GF_DIFF" -u "$TMPDIR/a.txt" "$TMPDIR/b.txt" | "$GF_GREP" "^[-+]" | "$GF_GREP" -v "^[-+][-+][-+]" | "$GF_WC" -l) || true
result=$(echo "$result" | tr -d ' ')
assert_eq "diff -u | grep changed lines" "2" "$result"

# ───────────────────────────────────
echo "--- Test 19: gf-tree | gf-grep ---"
# ディレクトリツリーから.goファイルを検索
result=$("$GF_TREE" "$TMPDIR/src" | "$GF_GREP" "\.go" | "$GF_WC" -l)
result=$(echo "$result" | tr -d ' ')
assert_eq "tree | grep .go | wc -l" "3" "$result"

# ───────────────────────────────────
echo "--- Test 20: echo | gf-xargs | gf-sort ---"
# xargsでechoした結果をソート
result=$(echo "cherry apple banana" | "$GF_XARGS" -n 1 echo | "$GF_SORT")
expected="apple
banana
cherry"
assert_eq "echo words | xargs -n 1 echo | sort" "$expected" "$result"

# ═══════════════════════════════════
echo ""
echo "=== Results ==="
echo "PASS: $PASS"
echo "FAIL: $FAIL"
echo "TOTAL: $((PASS + FAIL))"

if [ $FAIL -gt 0 ]; then
    echo ""
    echo "Failed tests:"
    echo -e "$ERRORS"
    exit 1
fi

echo ""
echo "All integration tests passed!"

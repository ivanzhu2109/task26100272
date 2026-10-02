package archive

import (
	"math"
	"testing"
)

const eps = 1e-12

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < eps
}

// 规格场景：查询 "a a"，规则 "a"->"a a" 与 "a a"->"a"，
// 消息 "a a a"、"a"、"b"，tf 必须分别为 3 和 1，df 必须为 2。
func TestSpecScenario(t *testing.T) {
	rules := []Rule{
		{From: []string{"a"}, To: []string{"a", "a"}},
		{From: []string{"a", "a"}, To: []string{"a"}},
	}
	msgs := []Message{
		{ID: 1, Tokens: []string{"a", "a", "a"}},
		{ID: 2, Tokens: []string{"a"}},
		{ID: 3, Tokens: []string{"b"}},
	}
	a := New(msgs, rules)

	// 展开集合应为 {a a, a a a, a a a a, a}：两个位置可各自或同时
	// 应用 "a"->"a a"，另可用 "a a"->"a"，去重后 4 个。
	exps := expandQuery([]string{"a", "a"}, rules)
	if len(exps) != 4 {
		t.Fatalf("expansions = %v, want 4 distinct", exps)
	}

	// tf：消息1 起点 {0,1,2} -> 3；消息2 起点 {0} -> 1；消息3 -> 0。
	if tf := countHitStarts(msgs[0].Tokens, exps); tf != 3 {
		t.Fatalf("tf(msg1) = %d, want 3", tf)
	}
	if tf := countHitStarts(msgs[1].Tokens, exps); tf != 1 {
		t.Fatalf("tf(msg2) = %d, want 1", tf)
	}
	if tf := countHitStarts(msgs[2].Tokens, exps); tf != 0 {
		t.Fatalf("tf(msg3) = %d, want 0", tf)
	}

	hits := a.Query([]string{"a", "a"})
	if len(hits) != 2 {
		t.Fatalf("df = %d, want 2 (hits=%v)", len(hits), hits)
	}

	// 排序：得分降序；消息1 tf 更高，应排前。
	if hits[0].ID != 1 || hits[1].ID != 2 {
		t.Fatalf("order = [%d %d], want [1 2]", hits[0].ID, hits[1].ID)
	}

	// 得分：N=3, df=2, avgdl=(3+1+1)/3=5/3。
	avgdl := 5.0 / 3.0
	want1 := bm25(3, 2, 3, 3, avgdl)
	want2 := bm25(3, 2, 1, 1, avgdl)
	if !almostEqual(hits[0].Score, want1) {
		t.Fatalf("score(msg1) = %v, want %v", hits[0].Score, want1)
	}
	if !almostEqual(hits[1].Score, want2) {
		t.Fatalf("score(msg2) = %v, want %v", hits[1].Score, want2)
	}

	// 与规格公式逐项核对。
	idf := math.Log(1 + (3.0-2.0+0.5)/(2.0+0.5))
	wantFormula1 := idf * 3 * 2.2 / (3 + 1.2*(0.25+0.75*3.0/avgdl))
	if !almostEqual(hits[0].Score, wantFormula1) {
		t.Fatalf("score(msg1) = %v, formula want %v", hits[0].Score, wantFormula1)
	}
}

// 替换结果恰好是另一规则左侧时，不得再次套用规则。
func TestNoChainedReplacement(t *testing.T) {
	rules := []Rule{
		{From: []string{"a"}, To: []string{"b"}},
		{From: []string{"b"}, To: []string{"c"}},
	}
	msgs := []Message{
		{ID: 1, Tokens: []string{"b"}},
		{ID: 2, Tokens: []string{"c"}},
		{ID: 3, Tokens: []string{"a"}},
	}
	a := New(msgs, rules)

	// 查询 "a" 的展开只能是 {a, b}，不能出现 c。
	exps := expandQuery([]string{"a"}, rules)
	if len(exps) != 2 {
		t.Fatalf("expansions = %v, want 2", exps)
	}

	hits := a.Query([]string{"a"})
	if len(hits) != 2 {
		t.Fatalf("df = %d, want 2 (hits=%v)", len(hits), hits)
	}
	for _, h := range hits {
		if h.ID == 2 {
			t.Fatalf("message 2 (c) must not hit; chained replacement applied")
		}
	}
}

// 规则不隐含反向关系。
func TestRulesAreDirectional(t *testing.T) {
	rules := []Rule{{From: []string{"a"}, To: []string{"b"}}}
	msgs := []Message{
		{ID: 1, Tokens: []string{"a"}},
		{ID: 2, Tokens: []string{"b"}},
	}
	a := New(msgs, rules)

	// 查询 "b" 只能命中消息2，不能反向命中消息1。
	hits := a.Query([]string{"b"})
	if len(hits) != 1 || hits[0].ID != 2 {
		t.Fatalf("hits = %v, want only message 2", hits)
	}
}

// 同一起点被多个展开命中、或命中不同长度片段，只计一次；
// 不同起点即使片段重叠也分别计数。
func TestTfDistinctStarts(t *testing.T) {
	rules := []Rule{
		{From: []string{"x"}, To: []string{"x", "y"}},
		{From: []string{"x"}, To: []string{"x", "y", "z"}},
	}
	exps := expandQuery([]string{"x"}, rules)
	// 展开 {x, x y, x y z} 在消息 "x y z" 上均从起点 0 命中 -> tf=1。
	if tf := countHitStarts([]string{"x", "y", "z"}, exps); tf != 1 {
		t.Fatalf("tf = %d, want 1 (same start counted once)", tf)
	}
	// 消息 "x y x y"：展开 x y 在起点 0 和 2 重叠命中 -> tf=2。
	if tf := countHitStarts([]string{"x", "y", "x", "y"}, exps); tf != 2 {
		t.Fatalf("tf = %d, want 2 (overlapping starts counted separately)", tf)
	}
}

// 重复规则与规则成环不改变一次替换语义，也不产生重复计分。
func TestDuplicateAndCyclicRules(t *testing.T) {
	rules := []Rule{
		{From: []string{"a"}, To: []string{"b"}},
		{From: []string{"a"}, To: []string{"b"}}, // 重复
		{From: []string{"b"}, To: []string{"a"}}, // 成环
	}
	exps := expandQuery([]string{"a"}, rules)
	if len(exps) != 2 { // {a, b}
		t.Fatalf("expansions = %v, want 2 distinct", exps)
	}

	msgs := []Message{
		{ID: 1, Tokens: []string{"b"}},
		{ID: 2, Tokens: []string{"a"}},
	}
	a := New(msgs, rules)
	hits := a.Query([]string{"a"})
	if len(hits) != 2 {
		t.Fatalf("df = %d, want 2", len(hits))
	}
	// 两条消息 tf 均为 1、dl 均为 1，得分应相同；并列时按编号升序。
	if !almostEqual(hits[0].Score, hits[1].Score) {
		t.Fatalf("scores differ: %v vs %v", hits[0].Score, hits[1].Score)
	}
	if hits[0].ID != 1 || hits[1].ID != 2 {
		t.Fatalf("tie order = [%d %d], want [1 2]", hits[0].ID, hits[1].ID)
	}
}

// 左侧在查询中重叠时，各种互不重叠的选择各自产生展开。
func TestOverlappingLeftSides(t *testing.T) {
	rules := []Rule{
		{From: []string{"a", "a"}, To: []string{"b"}},
	}
	exps := expandQuery([]string{"a", "a", "a"}, rules)
	// 展开：原查询、替换前两个、替换后两个，共 3 种。
	if len(exps) != 3 {
		t.Fatalf("expansions = %v, want 3", exps)
	}
	msgs := []Message{
		{ID: 1, Tokens: []string{"b", "a"}},
		{ID: 2, Tokens: []string{"a", "b"}},
		{ID: 3, Tokens: []string{"a", "a", "a"}},
	}
	a := New(msgs, rules)
	hits := a.Query([]string{"a", "a", "a"})
	if len(hits) != 3 {
		t.Fatalf("df = %d, want 3 (hits=%v)", len(hits), hits)
	}
}

// 多片段替换：同一查询中可选择多个互不重叠的片段同时替换。
func TestMultipleSegmentsReplaced(t *testing.T) {
	rules := []Rule{{From: []string{"a"}, To: []string{"x"}}}
	exps := expandQuery([]string{"a", "c", "a"}, rules)
	// 展开：a c a、x c a、a c x、x c x，共 4 种。
	if len(exps) != 4 {
		t.Fatalf("expansions = %v, want 4", exps)
	}
	msgs := []Message{{ID: 1, Tokens: []string{"x", "c", "x"}}}
	a := New(msgs, rules)
	hits := a.Query([]string{"a", "c", "a"})
	if len(hits) != 1 || hits[0].ID != 1 {
		t.Fatalf("hits = %v, want only message 1", hits)
	}
}

// 空档案查询返回空结果；无命中时 df 为 0。
func TestEmptyAndNoMatch(t *testing.T) {
	a := New(nil, nil)
	if hits := a.Query([]string{"a"}); len(hits) != 0 {
		t.Fatalf("empty archive hits = %v, want none", hits)
	}
	b := New([]Message{{ID: 1, Tokens: []string{"z"}}}, nil)
	if hits := b.Query([]string{"a"}); len(hits) != 0 {
		t.Fatalf("no-match hits = %v, want none", hits)
	}
}

// 档案构建后只读：调用方修改原切片不影响档案。
func TestArchiveIsImmutable(t *testing.T) {
	tokens := []string{"a"}
	from := []string{"a"}
	to := []string{"b"}
	a := New([]Message{{ID: 1, Tokens: tokens}}, []Rule{{From: from, To: to}})
	tokens[0] = "z"
	from[0] = "z"
	to[0] = "z"
	hits := a.Query([]string{"a"})
	if len(hits) != 1 || hits[0].ID != 1 {
		t.Fatalf("hits = %v, want message 1 (archive must be immutable)", hits)
	}
}

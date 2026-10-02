package synarchive

import (
	"math"
	"testing"
)

// 与题设公式一致的标准答案计算，用于独立核对 Search 的得分。
func wantScore(n, df, tf int, dl, avgdl float64) float64 {
	return math.Log(1+(float64(n)-float64(df)+0.5)/(float64(df)+0.5)) *
		float64(tf) * 2.2 / (float64(tf) + 1.2*(0.25+0.75*dl/avgdl))
}

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-12
}

// 题设保留场景一：查询 "a a"，规则 a->a a、a a->a，
// 消息 "a a a"、"a"、"b" 的 tf 必须为 3、1，df 必须为 2。
func TestSearchBasicScenario(t *testing.T) {
	rules := []Rule{
		{From: []string{"a"}, To: []string{"a", "a"}},
		{From: []string{"a", "a"}, To: []string{"a"}},
	}
	a := Build([]Message{
		{ID: 1, Tokens: []string{"a", "a", "a"}},
		{ID: 2, Tokens: []string{"a"}},
		{ID: 3, Tokens: []string{"b"}},
	}, rules)

	hits := a.Search([]string{"a", "a"})
	if len(hits) != 2 {
		t.Fatalf("命中数 = %d,  want 2 (df=2): %+v", len(hits), hits)
	}
	tfByID := map[int64]int{}
	for _, h := range hits {
		tfByID[h.ID] = h.TF
	}
	if tfByID[1] != 3 || tfByID[2] != 1 {
		t.Fatalf("tf = %v, want {1:3, 2:1}", tfByID)
	}
	if _, ok := tfByID[3]; ok {
		t.Fatalf("消息 3 不应命中: %+v", hits)
	}

	// 核对得分：N=3, df=2, avgdl=5/3。
	avgdl := 5.0 / 3.0
	for _, h := range hits {
		dl := 3.0
		if h.ID == 2 {
			dl = 1.0
		}
		want := wantScore(3, 2, h.TF, dl, avgdl)
		if !almostEqual(h.Score, want) {
			t.Fatalf("消息 %d 得分 = %v, want %v", h.ID, h.Score, want)
		}
	}
	// tf=3 的消息得分应高于 tf=1 的消息，排序降序。
	if hits[0].ID != 1 || hits[1].ID != 2 {
		t.Fatalf("排序 = %+v, want [1 2]", hits)
	}
}

// 题设保留场景二：替换结果恰好是另一规则左侧时，不得再次套用规则。
// 规则 a->b、b->c，查询 "a" 只能展开为 {a, b}，消息 "c" 不得命中。
func TestSearchNoCascade(t *testing.T) {
	rules := []Rule{
		{From: []string{"a"}, To: []string{"b"}},
		{From: []string{"b"}, To: []string{"c"}},
	}
	a := Build([]Message{
		{ID: 1, Tokens: []string{"a"}},
		{ID: 2, Tokens: []string{"b"}},
		{ID: 3, Tokens: []string{"c"}},
	}, rules)

	hits := a.Search([]string{"a"})
	if len(hits) != 2 {
		t.Fatalf("命中数 = %d, want 2: %+v", len(hits), hits)
	}
	for _, h := range hits {
		if h.ID == 3 {
			t.Fatalf("消息 3 (c) 不应命中：替换结果不得再次套用规则: %+v", hits)
		}
	}
}

// 得分相同时按编号升序；命中集合按得分降序。
func TestSearchOrdering(t *testing.T) {
	a := Build([]Message{
		{ID: 5, Tokens: []string{"x", "y"}},
		{ID: 2, Tokens: []string{"x", "y"}},
		{ID: 9, Tokens: []string{"z"}},
	}, nil)

	hits := a.Search([]string{"x", "y"})
	if len(hits) != 2 || hits[0].ID != 2 || hits[1].ID != 5 {
		t.Fatalf("同分应按编号升序 [2 5], got %+v", hits)
	}
}

// 不同起点即使片段重叠也分别计数；同一起点被多个展开命中只算一次。
func TestSearchOverlappingStarts(t *testing.T) {
	rules := []Rule{
		{From: []string{"a"}, To: []string{"a", "a"}}, // 同一起点可被 "a" 与 "a a" 同时命中
	}
	a := Build([]Message{
		{ID: 1, Tokens: []string{"a", "a", "a"}},
		{ID: 2, Tokens: []string{"b", "b"}},
	}, rules)

	hits := a.Search([]string{"a"})
	if len(hits) != 1 || hits[0].ID != 1 || hits[0].TF != 3 {
		t.Fatalf("重叠起点应分别计数 tf=3, got %+v", hits)
	}
}

// 无规则时仅原查询参与匹配；df 只统计至少命中一次的消息。
func TestSearchNoRules(t *testing.T) {
	a := Build([]Message{
		{ID: 1, Tokens: []string{"new", "york", "city"}},
		{ID: 2, Tokens: []string{"york", "new"}},
		{ID: 3, Tokens: []string{"new", "york"}},
	}, nil)

	hits := a.Search([]string{"new", "york"})
	if len(hits) != 2 {
		t.Fatalf("命中数 = %d, want 2: %+v", len(hits), hits)
	}
	for _, h := range hits {
		if h.ID == 2 {
			t.Fatalf("消息 2 词序不符，不应命中: %+v", hits)
		}
		if h.TF != 1 {
			t.Fatalf("消息 %d tf = %d, want 1", h.ID, h.TF)
		}
	}
	// N=3, df=2, avgdl=7/3，核对得分。
	avgdl := 7.0 / 3.0
	for _, h := range hits {
		dl := 3.0
		if h.ID == 3 {
			dl = 2.0
		}
		if want := wantScore(3, 2, 1, dl, avgdl); !almostEqual(h.Score, want) {
			t.Fatalf("消息 %d 得分 = %v, want %v", h.ID, h.Score, want)
		}
	}
}

// 档案统计（N、avgdl）不随展开或命中集合改变：展开全部落空时无结果，
// 且命中子集的 df 只按实际命中消息计。
func TestSearchStatsIndependentOfExpansions(t *testing.T) {
	rules := []Rule{
		{From: []string{"q"}, To: []string{"missing"}},
	}
	a := Build([]Message{
		{ID: 1, Tokens: []string{"q"}},
		{ID: 2, Tokens: []string{"other"}},
	}, rules)

	// 展开 {q, missing}，只有消息 1 命中，df=1，avgdl 恒为 1。
	hits := a.Search([]string{"q"})
	if len(hits) != 1 || hits[0].ID != 1 {
		t.Fatalf("命中 = %+v, want 仅消息 1", hits)
	}
	if want := wantScore(2, 1, 1, 1, 1); !almostEqual(hits[0].Score, want) {
		t.Fatalf("得分 = %v, want %v", hits[0].Score, want)
	}

	// 全部落空时返回空结果。
	if got := a.Search([]string{"missing"}); len(got) != 0 {
		t.Fatalf("应无命中, got %+v", got)
	}
}

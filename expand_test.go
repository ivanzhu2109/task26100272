package synarchive

import (
	"sort"
	"strings"
	"testing"
)

func expansionsAsStrings(exps [][]string) []string {
	out := make([]string, len(exps))
	for i, e := range exps {
		out[i] = strings.Join(e, " ")
	}
	sort.Strings(out)
	return out
}

func equalSets(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// 题设保留场景：查询 "a a"，规则 a->a a 与 a a->a。
// 展开集合应为 {"a a", "a", "a a a", "a a a a"}，无重复。
func TestExpandBasicScenario(t *testing.T) {
	rules := []Rule{
		{From: []string{"a"}, To: []string{"a", "a"}},
		{From: []string{"a", "a"}, To: []string{"a"}},
	}
	got := expansionsAsStrings(expand([]string{"a", "a"}, rules))
	want := []string{"a", "a a", "a a a", "a a a a"}
	sort.Strings(want)
	if !equalSets(got, want) {
		t.Fatalf("expand(a a) = %v, want %v", got, want)
	}
}

// 替换结果恰好是另一规则的左侧时，不得再次套用规则（不成链）。
func TestExpandNoCascade(t *testing.T) {
	rules := []Rule{
		{From: []string{"a"}, To: []string{"b"}},
		{From: []string{"b"}, To: []string{"c"}},
	}
	got := expansionsAsStrings(expand([]string{"a"}, rules))
	want := []string{"a", "b"}
	sort.Strings(want)
	if !equalSets(got, want) {
		t.Fatalf("expand(a) = %v, want %v（不得级联出 c）", got, want)
	}
}

// 规则成环、重复规则、左侧重叠均不改变一次替换语义，且不产生重复展开。
func TestExpandCyclesDuplicatesOverlaps(t *testing.T) {
	rules := []Rule{
		{From: []string{"a"}, To: []string{"b"}},
		{From: []string{"b"}, To: []string{"a"}}, // 成环
		{From: []string{"a"}, To: []string{"b"}}, // 重复
		{From: []string{"a", "b"}, To: []string{"c"}},
		{From: []string{"b", "a"}, To: []string{"d"}}, // 与上一条左侧重叠
	}
	got := expansionsAsStrings(expand([]string{"a", "b", "a"}, rules))
	// 手工枚举：每个位置保留或替换，互不重叠，仅作用于原查询。
	want := []string{
		"a b a", // 原查询
		"b b a", // 位置0: a->b
		"a a a", // 位置1: b->a
		"a b b", // 位置2: a->b
		"b b b", // 位置0、2 同时替换
		"b a a", // 位置0、1 同时替换
		"a a b", // 位置1、2 同时替换
		"b a b", // 三个位置都替换
		"c a",   // 片段[0,2): a b->c
		"c b",   // 片段[0,2): a b->c，且位置2: a->b
		"a d",   // 片段[1,3): b a->d
		"b d",   // 位置0: a->b，且片段[1,3): b a->d
	}
	sort.Strings(want)
	if !equalSets(got, want) {
		t.Fatalf("expand(a b a) = %v, want %v", got, want)
	}
}

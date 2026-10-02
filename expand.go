package synarchive

// expand 对原始查询做一次替换语义下的全部合法展开（含不替换的原查询）。
//
// 语义：可以选择原查询中任意个互不重叠的片段，每个片段与某条规则的左侧
// 完全相同，并替换为该规则右侧；替换只作用于原查询，展开结果不再套用规则。
// 规则可重复、可成环、左侧可在查询中重叠，均不改变上述语义。
// 返回值为去重后的展开集合。
func expand(query []string, rules []Rule) [][]string {
	seen := make(map[string]struct{})
	var out [][]string

	var rec func(pos int, acc []string)
	rec = func(pos int, acc []string) {
		if pos == len(query) {
			key := joinTokens(acc)
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				cp := make([]string, len(acc))
				copy(cp, acc)
				out = append(out, cp)
			}
			return
		}
		// 选择一：保留当前词元。
		rec(pos+1, append(acc, query[pos]))
		// 选择二：若某条规则左侧从 pos 开始完全匹配，则替换为右侧。
		for _, r := range rules {
			if matchAt(query, pos, r.From) {
				rec(pos+len(r.From), append(acc, r.To...))
			}
		}
	}
	rec(0, nil)
	return out
}

// matchAt 判断 pat 是否与 tokens 从 start 开始的连续片段完全一致。
func matchAt(tokens []string, start int, pat []string) bool {
	if start+len(pat) > len(tokens) {
		return false
	}
	for i, p := range pat {
		if tokens[start+i] != p {
			return false
		}
	}
	return true
}

// joinTokens 用不可打印分隔符连接词元，作为集合键，避免词元本身含空格时歧义。
func joinTokens(tokens []string) string {
	key := ""
	for i, t := range tokens {
		if i > 0 {
			key += "\x00"
		}
		key += t
	}
	return key
}

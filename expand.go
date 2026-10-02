package archive

// expandQuery 对原查询做一次同义替换展开：保留原查询本身，
// 并可选择任意个互不重叠、与某条规则左侧完全相同的原查询片段，
// 各自替换为对应右侧。替换只作用于原查询，替换结果不再套用规则；
// 重复规则、规则成环、左侧在查询中重叠均不改变这一语义。
// 返回去重后的全部合法展开（含原查询）。
func expandQuery(query []string, rules []Rule) [][]string {
	seen := make(map[string]struct{})
	var expansions [][]string

	var walk func(pos int, out []string)
	walk = func(pos int, out []string) {
		if pos == len(query) {
			key := joinTokens(out)
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				expansions = append(expansions, out)
			}
			return
		}
		// 保留当前词元，不做替换。
		walk(pos+1, append(append([]string{}, out...), query[pos]))
		// 尝试以 pos 为起点的每条规则；替换片段互不重叠由
		// “替换后跳到片段末尾”保证，重叠左侧自然产生不同展开。
		for _, r := range rules {
			if matchAt(query, pos, r.From) {
				next := append(append([]string{}, out...), r.To...)
				walk(pos+len(r.From), next)
			}
		}
	}
	walk(0, nil)
	return expansions
}

// matchAt 判断 tokens 从 pos 开始的连续片段是否与 pattern 完全相同。
func matchAt(tokens []string, pos int, pattern []string) bool {
	if pos+len(pattern) > len(tokens) {
		return false
	}
	for i, p := range pattern {
		if tokens[pos+i] != p {
			return false
		}
	}
	return true
}

// joinTokens 用不可打印分隔符拼接词元，作为去重键。
func joinTokens(tokens []string) string {
	key := ""
	for _, t := range tokens {
		key += t + "\x00"
	}
	return key
}

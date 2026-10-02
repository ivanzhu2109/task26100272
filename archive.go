// Package synarchive 提供进程内的客服消息档案同义短语检索能力。
//
// 上层已完成切词与大小写统一；消息、查询与规则两侧均为非空词元序列，
// 词元按完整字符串比较。同义规则是有方向的短语替换，仅作用于查询侧，
// 且只做一次替换（展开结果不再套用规则）。整个查询短语视为一个检索项，
// 使用 BM25 评分。
package synarchive

import "sort"

// Message 是一条客户沟通记录，ID 唯一，Tokens 为已完成切词与大小写统一的词元序列。
type Message struct {
	ID     int64
	Tokens []string
}

// Rule 是一条有方向的同义替换规则：查询中连续出现 From 时可替换为 To。
type Rule struct {
	From []string
	To   []string
}

// Hit 是一条命中消息的检索结果。
type Hit struct {
	ID    int64   // 消息编号
	Score float64 // BM25 得分
	TF    int     // 该消息中不同命中起点的数量
}

// Archive 是只读的消息检索档案，数据驻留进程内。
type Archive struct {
	messages []Message
	rules    []Rule
	n        int
	avgdl    float64
}

// Build 以给定消息与同义规则建立只读档案。统计信息在建档时固定，
// 不随查询展开或命中集合改变。
func Build(messages []Message, rules []Rule) *Archive {
	a := &Archive{
		messages: messages,
		rules:    rules,
		n:        len(messages),
	}
	total := 0
	for _, m := range messages {
		total += len(m.Tokens)
	}
	if a.n > 0 {
		a.avgdl = float64(total) / float64(a.n)
	}
	return a
}

// Search 以单个查询短语检索档案，返回按得分降序、得分相同时按编号升序的命中结果。
func (a *Archive) Search(query []string) []Hit {
	expansions := expand(query, a.rules)

	// 统计每条消息的 tf：不同命中起点的数量。
	// 同一起点被多个展开命中或命中不同长度片段只算一次；
	// 不同起点即使片段重叠也分别计数。
	tfs := make([]int, len(a.messages))
	df := 0
	for i, m := range a.messages {
		tf := 0
		for start := 0; start < len(m.Tokens); start++ {
			for _, e := range expansions {
				if matchAt(m.Tokens, start, e) {
					tf++
					break // 同一起点只计一次
				}
			}
		}
		tfs[i] = tf
		if tf > 0 {
			df++
		}
	}

	hits := make([]Hit, 0, df)
	for i, m := range a.messages {
		if tfs[i] == 0 {
			continue
		}
		hits = append(hits, Hit{
			ID:    m.ID,
			Score: bm25(a.n, df, tfs[i], float64(len(m.Tokens)), a.avgdl),
			TF:    tfs[i],
		})
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		return hits[i].ID < hits[j].ID
	})
	return hits
}

// Package archive 提供客服消息档案的同义短语检索能力。
// 数据驻留进程内，档案构建后只读，支持单个短语查询。
package archive

import "sort"

// Message 是一条客户沟通记录。ID 在档案内唯一，Tokens 为已切分、
// 已统一大小写的非空词元序列。
type Message struct {
	ID     int
	Tokens []string
}

// Rule 是一条有方向的同义替换规则：查询中连续出现 From 时，
// 可整体替换为 To。规则不隐含反向关系。
type Rule struct {
	From []string
	To   []string
}

// Archive 是只读的消息档案，构建后可并发查询。
type Archive struct {
	messages []Message
	rules    []Rule
	avgdl    float64
}

// New 以给定消息与同义规则建立只读档案。
// 消息与规则切片会被拷贝，调用方后续修改不影响档案。
func New(messages []Message, rules []Rule) *Archive {
	ms := make([]Message, len(messages))
	total := 0
	for i, m := range messages {
		tokens := append([]string(nil), m.Tokens...)
		ms[i] = Message{ID: m.ID, Tokens: tokens}
		total += len(tokens)
	}
	rs := make([]Rule, len(rules))
	for i, r := range rules {
		rs[i] = Rule{
			From: append([]string(nil), r.From...),
			To:   append([]string(nil), r.To...),
		}
	}
	avgdl := 0.0
	if len(ms) > 0 {
		avgdl = float64(total) / float64(len(ms))
	}
	return &Archive{messages: ms, rules: rs, avgdl: avgdl}
}

// Hit 是一条命中结果，包含消息编号与 BM25 相关性得分。
type Hit struct {
	ID    int
	Score float64
}

// Query 以单个短语（非空词元序列）检索档案。
// 某个合法展开与消息中的连续词元片段完全一致即命中；
// tf 为消息中不同命中起点数，df 为至少命中一次的消息总数。
// 返回按得分降序、得分相同按编号升序排列的命中结果。
func (a *Archive) Query(phrase []string) []Hit {
	expansions := expandQuery(phrase, a.rules)

	tfs := make([]int, len(a.messages))
	df := 0
	for i, m := range a.messages {
		tf := countHitStarts(m.Tokens, expansions)
		tfs[i] = tf
		if tf > 0 {
			df++
		}
	}

	n := len(a.messages)
	hits := make([]Hit, 0, df)
	for i, m := range a.messages {
		if tfs[i] == 0 {
			continue
		}
		hits = append(hits, Hit{
			ID:    m.ID,
			Score: bm25(n, df, tfs[i], len(m.Tokens), a.avgdl),
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

// countHitStarts 统计消息中被任一展开命中的不同起点数量。
// 同一起点被多个展开命中或命中不同长度片段只算一次；
// 不同起点即使片段重叠也分别计数。
func countHitStarts(tokens []string, expansions [][]string) int {
	starts := make(map[int]struct{})
	for _, exp := range expansions {
		for pos := 0; pos+len(exp) <= len(tokens); pos++ {
			if matchAt(tokens, pos, exp) {
				starts[pos] = struct{}{}
			}
		}
	}
	return len(starts)
}

package archive

import "math"

// BM25 常用参数。
const (
	k1 = 1.2
	b  = 0.75
)

// bm25 计算整个查询短语作为单一检索项的 BM25 得分。
// n 为档案消息总数，df 为至少命中一次的消息数，
// tf 为当前消息中不同命中起点数，dl 为当前消息原始词元数，
// avgdl 为全部消息的平均词元数。以上统计均不随展开或命中集合改变。
func bm25(n, df, tf, dl int, avgdl float64) float64 {
	idf := math.Log(1 + (float64(n)-float64(df)+0.5)/(float64(df)+0.5))
	tff := float64(tf) * (k1 + 1) / (float64(tf) + k1*(1-b+b*float64(dl)/avgdl))
	return idf * tff
}

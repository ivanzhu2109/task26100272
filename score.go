package synarchive

import "math"

// bm25 按题设公式计算整个查询短语作为单个检索项的得分：
//
//	ln(1+(N-df+0.5)/(df+0.5)) * tf*2.2 / (tf + 1.2*(0.25+0.75*dl/avgdl))
//
// N、dl、avgdl 为档案固有统计，不随展开或命中集合改变。
func bm25(n, df, tf int, dl, avgdl float64) float64 {
	idf := math.Log(1 + (float64(n)-float64(df)+0.5)/(float64(df)+0.5))
	tff := float64(tf)
	return idf * tff * 2.2 / (tff + 1.2*(0.25+0.75*dl/avgdl))
}

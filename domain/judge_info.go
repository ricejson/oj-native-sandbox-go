package domain

// JudgeInfo 判题信息
type JudgeInfo struct {
	Message string `json:"message"` // 执行信息
	Memory  int64  `json:"memory"`  // 消耗内存（单位KB）
	Time    int64  `json:"time"`    // 消耗时间（单位ms）
}

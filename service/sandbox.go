package service

import (
	"context"
	"time"

	"github.com/ricejson/oj-native-sandbox-go/domain"
)

const (
	VolumeDir      = "/app"
	BaseDir        = "./tmpcode"
	SourceFileName = "main.go"
	BuiltFileName  = "main"
	TimeOut        = 5 * time.Second
)

type ExecuteCodeRequest struct {
	Code         string   `json:"code"`          // 代码
	Language     string   `json:"language"`      // 编程语言
	InputSamples []string `json:"input_samples"` // 输入样例
}

type ExecuteCodeResponse struct {
	OutputResults []string          `json:"output_results"` // 输出结果
	JudgeInfo     *domain.JudgeInfo `json:"judge_info"`     // 判题信息
}

// CodeSandbox 代码沙箱
type CodeSandbox interface {
	ExecuteCode(ctx context.Context, req *ExecuteCodeRequest) (*ExecuteCodeResponse, error)
}

package service

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ricejson/gotool/logx"
	"github.com/ricejson/oj-native-sandbox-go/common/consts"
	"github.com/ricejson/oj-native-sandbox-go/domain"
)

const (
	BaseDir        = "./tmpcode"
	SourceFileName = "main.go"
	BuiltFileName  = "main"
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

type NativeCodeSandbox struct {
	logger logx.Logger
}

func NewNativeCodeSandbox(logger logx.Logger) *NativeCodeSandbox {
	return &NativeCodeSandbox{logger: logger}
}

func (s *NativeCodeSandbox) ExecuteCode(ctx context.Context, req *ExecuteCodeRequest) (*ExecuteCodeResponse, error) {
	// 1. 将用户代码写入文件
	code := req.Code
	err := os.MkdirAll(BaseDir, os.ModePerm)
	if err != nil {
		return nil, err
	}
	userPath := BaseDir + string(os.PathSeparator) + uuid.NewString()
	err = os.MkdirAll(userPath, os.ModePerm)
	if err != nil {
		return nil, err
	}
	// 销毁资源
	defer os.RemoveAll(userPath)
	userSourceFile := userPath + string(os.PathSeparator) + SourceFileName
	err = os.WriteFile(userSourceFile, []byte(code), os.ModePerm)
	if err != nil {
		return nil, err
	}
	// 2. 执行命令
	userBuiltFileName := userPath + string(os.PathSeparator) + BuiltFileName
	buildCmd := exec.Command("go", "build", "-o", userBuiltFileName, userPath)
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		s.logger.Error("build fail，output:", logx.Error(errors.New(strings.TrimSpace(string(output)))))
		return &ExecuteCodeResponse{
			OutputResults: nil,
			JudgeInfo: &domain.JudgeInfo{
				Message: consts.JudgeMessageCompileError,
				Time:    -1,
				Memory:  -1,
			},
		}, err
	}
	s.logger.Info("build success，output:", logx.String("output", strings.TrimSpace(string(output))))

	// 3. 运行可执行文件
	inputSamples := req.InputSamples
	outputResults := make([]string, 0, len(inputSamples))
	timeArr := make([]int64, 0, len(inputSamples))
	for _, inputSample := range inputSamples {
		runCmd := exec.Command(userBuiltFileName, strings.Split(inputSample, " ")...)
		startTime := time.Now()
		output, err = runCmd.CombinedOutput()
		timeArr = append(timeArr, time.Since(startTime).Milliseconds())
		if err != nil {
			s.logger.Error("run fail，output:", logx.Error(errors.New(strings.TrimSpace(string(output)))))
			return &ExecuteCodeResponse{
				OutputResults: outputResults,
				JudgeInfo: &domain.JudgeInfo{
					Message: consts.JudgeMessageRuntimeError,
					Memory:  -1,
					Time:    -1,
				},
			}, err
		}
		s.logger.Info("run success，output:", logx.String("output", strings.TrimSpace(string(output))))
		outputResults = append(outputResults, strings.TrimSpace(string(output)))
	}
	// 4. 组装控制台返回信息
	// 计算时间取最大值
	maxTime := int64(-1)
	for _, t := range timeArr {
		maxTime = max(maxTime, t)
	}
	return &ExecuteCodeResponse{
		OutputResults: outputResults,
		JudgeInfo: &domain.JudgeInfo{
			Message: consts.JudgeMessageAccept,
			Memory:  0,
			Time:    maxTime,
		},
	}, nil
}

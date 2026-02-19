package service

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/google/uuid"
	"github.com/moby/moby/api/types/mount"
	"github.com/ricejson/gotool/logx"
	"github.com/ricejson/oj-native-sandbox-go/common/consts"
	"github.com/ricejson/oj-native-sandbox-go/domain"
)

type DockerSandbox struct {
	logger       logx.Logger
	dockerClient *docker.Client
	imagePulled  bool
}

func NewDockerCodeSandbox(logger logx.Logger, dockerClient *docker.Client) *DockerSandbox {
	return &DockerSandbox{
		logger:       logger,
		dockerClient: dockerClient,
	}
}

func (s *DockerSandbox) ExecuteCode(ctx context.Context, req *ExecuteCodeRequest) (*ExecuteCodeResponse, error) {
	// 1. 将用户代码写入文件
	code := req.Code
	err := os.MkdirAll(BaseDir, os.ModePerm)
	if err != nil {
		return nil, err
	}
	uuidDir := uuid.NewString()
	userPath := BaseDir + string(os.PathSeparator) + uuidDir
	volumeUserPath := VolumeDir + string(os.PathSeparator) + uuidDir
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

	// 3. 运行可执行文件（docker环境）
	// 拉取镜像
	if !s.imagePulled {
		err = s.dockerClient.PullImage(docker.PullImageOptions{
			Repository: "golang",      // 镜像名称
			Tag:        "1.25-alpine", // 镜像标签
		}, docker.AuthConfiguration{})
		if err == nil {
			s.imagePulled = true
		}
	}
	// 创建容器
	path, _ := filepath.Abs(BaseDir)
	container, err := s.dockerClient.CreateContainer(docker.CreateContainerOptions{
		HostConfig: &docker.HostConfig{
			Mounts: []docker.HostMount{
				{
					Type:   string(mount.TypeBind),
					Source: path,
					Target: "/app",
				},
			},
		},
		Config: &docker.Config{
			Image:        "golang:1.25-alpine",
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
			Tty:          true,
		},
	})
	if err != nil {
		return nil, err
	}
	// 启动容器
	err = s.dockerClient.StartContainer(container.ID, nil)
	if err != nil {
		return nil, err
	}
	inputSamples := req.InputSamples
	outputResults := make([]string, 0, len(inputSamples))
	timeArr := make([]int64, 0, len(inputSamples))
	memoryArr := make([]int64, 0, len(inputSamples))
	for _, inputSample := range inputSamples {
		// 创建执行命令
		execCmd := []string{"go", "run", volumeUserPath + string(os.PathSeparator) + SourceFileName}
		execCmd = append(execCmd, strings.Split(inputSample, " ")...)
		createExec, err := s.dockerClient.CreateExec(docker.CreateExecOptions{
			Container:    container.ID,
			Cmd:          execCmd,
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
		})
		if err != nil {
			return nil, err
		}
		// 启动内存监控协程
		memoryChan := make(chan int64, 100)
		stopChan := make(chan struct{}, 1)
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			setStatMonitor(s.dockerClient, container.ID, memoryChan, stopChan)
		}()
		// 执行命令
		startTime := time.Now()
		var stdout, stderr bytes.Buffer
		err = s.dockerClient.StartExec(createExec.ID, docker.StartExecOptions{
			OutputStream: &stdout,
			ErrorStream:  &stderr,
		})
		close(stopChan)
		wg.Wait()
		close(memoryChan)
		timeArr = append(timeArr, time.Since(startTime).Milliseconds())
		var maxMemory int64 = -1
		for mem := range memoryChan {
			if mem > maxMemory {
				maxMemory = mem
			}
		}
		memoryArr = append(memoryArr, maxMemory)
		if err != nil {
			s.logger.Error("run fail，output:", logx.Error(errors.New(strings.TrimSpace(stderr.String()))))
			return &ExecuteCodeResponse{
				OutputResults: outputResults,
				JudgeInfo: &domain.JudgeInfo{
					Message: consts.JudgeMessageRuntimeError,
					Memory:  -1,
					Time:    -1,
				},
			}, err
		}
		s.logger.Info("run success，output:", logx.String("output", strings.TrimSpace(stdout.String())))
		outputResults = append(outputResults, strings.TrimSpace(stdout.String()))
	}
	// 4. 组装控制台返回信息
	// 计算时间取最大值
	maxTime := int64(-1)
	for _, t := range timeArr {
		maxTime = max(maxTime, t)
	}
	maxMemory := int64(-1)
	for _, m := range memoryArr {
		maxTime = max(maxMemory, m)
	}
	return &ExecuteCodeResponse{
		OutputResults: outputResults,
		JudgeInfo: &domain.JudgeInfo{
			Message: consts.JudgeMessageAccept,
			Memory:  maxMemory,
			Time:    maxTime,
		},
	}, nil
}

func setStatMonitor(dockerClient *docker.Client, containerId string, memoryChan chan<- int64, stopChan <-chan struct{}) {
	ticker := time.NewTicker(time.Millisecond * 1)
	defer ticker.Stop()
	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			stats := make(chan *docker.Stats, 1)
			err := dockerClient.Stats(docker.StatsOptions{
				ID:     containerId,
				Stats:  stats,
				Stream: false,
			})
			if err != nil {
				continue
			}
			stat := <-stats
			memory := stat.MemoryStats.Usage
			if memory > 0 {
				memoryChan <- int64(stat.MemoryStats.Usage)
			}
		}
	}
}

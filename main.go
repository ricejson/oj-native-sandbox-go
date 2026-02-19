package main

import (
	"context"
	"fmt"
	"os"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/ricejson/gotool/logx"
	"github.com/ricejson/oj-native-sandbox-go/service"
)

func main() {
	logger := logx.NewZapLogger()
	//s := service.NewNativeCodeSandbox(logger)
	client, er := docker.NewClientFromEnv()
	if er != nil {
		panic(er)
	}
	s := service.NewDockerCodeSandbox(logger, client)
	//bytes, _ := os.ReadFile("./samples/timeerr/main.go")
	bytes, _ := os.ReadFile("./samples/main.go")
	timeErrCodeStr := string(bytes)
	resp, err := s.ExecuteCode(context.Background(), &service.ExecuteCodeRequest{
		timeErrCodeStr,
		"Go",
		[]string{"1 2"},
	})
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(resp)
}

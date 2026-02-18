package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ricejson/gotool/logx"
	"github.com/ricejson/oj-native-sandbox-go/service"
)

func main() {
	logger := logx.NewZapLogger()
	s := service.NewNativeCodeSandbox(logger)
	//bytes, _ := os.ReadFile("./samples/timeerr/main.go")
	bytes, _ := os.ReadFile("./samples/memoryerr/main.go")
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

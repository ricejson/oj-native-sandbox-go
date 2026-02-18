package main

import (
	"context"

	"github.com/ricejson/oj-native-sandbox-go/service"
)

func main() {
	s := &service.NativeCodeSandbox{}
	_, err := s.ExecuteCode(context.Background(), &service.ExecuteCodeRequest{
		"package main\n\nimport (\n\"fmt\"\n\"os\"\n)\n\nfunc main() {\n\ta := os.Args[1]\n\tb := os.Args[2]\n\tfmt.Println(a + b)\n}",
		"Go",
		[]string{"1 2", "3 4"},
	})
	if err != nil {
		panic(err)
	}
}

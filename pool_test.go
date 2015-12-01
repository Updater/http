package http_test

import (
	"fmt"

	"github.com/Bridgevine/t-http"
)

func ExampleNewClientPool() {
	cp := http.NewClientPool()

	client := cp.GetClient(10) //Requests a http client with 10ns timeout
	fmt.Println(client.Timeout)

	// Output:
	// 10ns
}

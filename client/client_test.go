package client

import (
	"fmt"
)

func ExampleExamples_Query() {
	c := NewClient("localhost:8002", "localhost:8003")
	messages := c.Query("select *;")
	fmt.Printf("got %v messages\n", len(*messages))
}

func ExampleExamples_Subscribe() {
	c := NewClient("localhost:8002", "localhost:8003")
	channel := c.Subscribe("has uuid;")
	for m := range channel {
		fmt.Println(m)
	}
}

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"jsonclient"
)

type Post struct {
	UserID int    `json:"userId"`
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

func main() {
	// Use JSONPlaceholder as a fake service.
	client := jsonclient.NewClient(
		"https://jsonplaceholder.typicode.com",
		jsonclient.WithUserAgent("MyApp/1.0"),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var post Post
	if err := client.Get(ctx, "/posts/1", &post); err != nil {
		log.Fatalf("GET error: %v", err)
	}
	fmt.Printf("Post: %+v\n", post)

	newPost := map[string]interface{}{
		"title":  "foo",
		"body":   "bar",
		"userId": 1,
	}
	var createdPost Post
	if err := client.Post(ctx, "/posts", &newPost, &createdPost); err != nil {
		log.Fatalf("POST error: %v", err)
	}
	fmt.Printf("Created: %+v\n", createdPost)
}

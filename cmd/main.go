package main

import (
	"chat-session/internal/router"
	"chat-session/internal/session"
	"fmt"
	"net/http"
)

func main() {
	//init repository
	//	messageRepo := repository.NewMessage()

	//init service
	s := session.NewService()

	//init router
	r := router.InitRouter(s)

	//start service
	fmt.Println("start on port :9000")
	err := http.ListenAndServe(":9000", r)
	if err != nil {
		panic(err)
	}
}

package main

import (
	"chat-session/internal/config"
	"chat-session/internal/repository"
	"chat-session/internal/router"
	"chat-session/internal/session"
	"go.uber.org/zap"
	"net/http"
)

func main() {
	//init config
	cfg := config.InitConfig()
	defer cfg.Free()

	//init repository
	messageRepo := repository.NewMessage(cfg.DB)

	//init service
	s := session.NewService(messageRepo)

	//init router
	r := router.InitRouter(s)

	//start service
	zap.S().Infof("start on %v", cfg.Env.Port)
	err := http.ListenAndServe(cfg.Env.Port, r)
	if err != nil {
		panic(err)
	}
}

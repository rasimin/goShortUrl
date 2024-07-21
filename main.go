package main

import (
	"goshorturl/config"
	"goshorturl/service"
	"goshorturl/setup"
)

func main() {
	cfg := config.LoadConfig()

	db := setup.SetupDatabase(cfg.DatabaseDSN)
	redisClient := setup.SetupRedis(cfg.RedisAddr)
	urlService := service.NewURLService(db, redisClient)

	go setup.StartGRPCServer(urlService, cfg.GRPCPort)
	setup.StartHTTPServer(urlService, cfg.HTTPPort, cfg.GRPCPort)
}

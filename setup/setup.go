package setup

import (
	"context"
	"log"
	"net"

	"goshorturl/entity"
	"goshorturl/proto"
	"goshorturl/service"

	"github.com/gin-gonic/gin"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func SetupDatabase(dsn string) *gorm.DB {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database: ", err)
	}

	err = db.AutoMigrate(&entity.URL{})
	if err != nil {
		log.Fatal("failed to migrate database: ", err)
	}
	return db
}

func SetupRedis(addr string) *redis.Client {
	redisClient := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	_, err := redisClient.Ping(context.Background()).Result()
	if err != nil {
		log.Fatal("failed to connect to Redis: ", err)
	}
	return redisClient
}

func StartGRPCServer(urlService *service.URLService, grpcPort string) {
	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterURLServiceServer(grpcServer, urlService)
	log.Printf("gRPC server listening at %v", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func StartHTTPServer(urlService *service.URLService, httpPort, grpcPort string) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err := proto.RegisterURLServiceHandlerFromEndpoint(ctx, mux, grpcPort, opts)
	if err != nil {
		log.Fatalf("failed to register gRPC gateway: %v", err)
	}

	gwServer := gin.Default()
	gwServer.GET("/:shorturl", urlService.RedirectURL)
	gwServer.Any("/v1/*grpc_gateway", gin.WrapH(mux))
	log.Printf("HTTP server running on port %s", httpPort)
	if err := gwServer.Run(httpPort); err != nil {
		log.Fatalf("failed to run HTTP server: %v", err)
	}
}

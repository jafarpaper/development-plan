package main

import (
	"log"
	"net"

	"go-clean-arango/configs"
	"go-clean-arango/internal/infrastructure/arango"
	"go-clean-arango/internal/usecase"
)

func main() {
	cfg := configs.LoadArangoConfig()

	arangoClient, err := arango.NewArangoClient(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to Arango: %v", err)
	}

	repo := arango.NewArangoUserRepo(arangoClient.DB)
	userUC := &usecase.UserUsecase{Repo: repo}

	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcSrv := grpc.NewGRPCServer(userUC)

	log.Println("gRPC server listening on :50051")
	if err := grpcSrv.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

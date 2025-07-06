package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	_ "io"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "activity-log-service/pkg/proto"
)

func main() {
	var (
		addr   = flag.String("addr", "localhost:9000", "gRPC server address")
		method = flag.String("method", "create", "Method to call: create, get, list")
		file   = flag.String("file", "", "JSON file with request data")
	)
	flag.Parse()

	// Connect to gRPC server
	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewActivityLogServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	switch *method {
	case "create":
		createActivityLog(ctx, client, *file)
	case "get":
		getActivityLog(ctx, client, *file)
	case "list":
		listActivityLogs(ctx, client, *file)
	default:
		log.Fatalf("Unknown method: %s", *method)
	}
}

func createActivityLog(ctx context.Context, client pb.ActivityLogServiceClient, file string) {
	if file == "" {
		file = "requests/grpc/create_activity_log.json"
	}

	data, err := os.ReadFile(file)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	var req pb.CreateActivityLogRequest
	if err := json.Unmarshal(data, &req); err != nil {
		log.Fatalf("Failed to unmarshal request: %v", err)
	}

	resp, err := client.CreateActivityLog(ctx, &req)
	if err != nil {
		log.Fatalf("CreateActivityLog failed: %v", err)
	}

	fmt.Printf("Created Activity Log:\n")
	printActivityLog(resp.ActivityLog)
}

func getActivityLog(ctx context.Context, client pb.ActivityLogServiceClient, file string) {
	if file == "" {
		file = "requests/grpc/get_activity_log.json"
	}

	data, err := os.ReadFile(file)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	var req pb.GetActivityLogRequest
	if err := json.Unmarshal(data, &req); err != nil {
		log.Fatalf("Failed to unmarshal request: %v", err)
	}

	resp, err := client.GetActivityLog(ctx, &req)
	if err != nil {
		log.Fatalf("GetActivityLog failed: %v", err)
	}

	fmt.Printf("Activity Log:\n")
	printActivityLog(resp.ActivityLog)
}

func listActivityLogs(ctx context.Context, client pb.ActivityLogServiceClient, file string) {
	if file == "" {
		file = "requests/grpc/list_activity_logs.json"
	}

	data, err := os.ReadFile(file)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	var req pb.ListActivityLogsRequest
	if err := json.Unmarshal(data, &req); err != nil {
		log.Fatalf("Failed to unmarshal request: %v", err)
	}

	resp, err := client.ListActivityLogs(ctx, &req)
	if err != nil {
		log.Fatalf("ListActivityLogs failed: %v", err)
	}

	fmt.Printf("Activity Logs (Total: %d, Page: %d, Limit: %d):\n", resp.Total, resp.Page, resp.Limit)
	for i, log := range resp.ActivityLogs {
		fmt.Printf("\n--- Activity Log %d ---\n", i+1)
		printActivityLog(log)
	}
}

func printActivityLog(log *pb.ActivityLog) {
	fmt.Printf("ID: %s\n", log.Id)
	fmt.Printf("Activity Name: %s\n", log.ActivityName)
	fmt.Printf("Company ID: %s\n", log.CompanyId)
	fmt.Printf("Object: %s (ID: %s)\n", log.ObjectName, log.ObjectId)
	fmt.Printf("Changes: %s\n", log.Changes)
	fmt.Printf("Message: %s\n", log.FormattedMessage)
	fmt.Printf("Actor: %s (%s) - %s\n", log.ActorName, log.ActorId, log.ActorEmail)
	if log.CreatedAt != nil {
		fmt.Printf("Created At: %s\n", log.CreatedAt.AsTime().Format(time.RFC3339))
	}
}

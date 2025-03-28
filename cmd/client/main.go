package main

import (
	"context"
	"github.com/alhaos/service/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"log"
	"time"
)

func main() {
	conn, err := grpc.NewClient(
		"82.202.140.217:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {
			panic(err)
		}
	}(conn)

	client := proto.NewCounterClient(conn)

	// Сначала логинимся
	loginResp, err := client.Login(context.Background(), &proto.LoginRequest{
		Username: "admin",
		Password: "password",
	})
	if err != nil {
		log.Fatalf("could not login: %v", err)
	}
	log.Printf("Login successful, token: %s", loginResp.Token)

	// Создаем контекст с токеном
	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.Pairs("authorization", loginResp.Token),
	)

	// Вызываем Increment с авторизацией
	for i := 0; i < 5; i++ {
		ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		resp, err := client.Increment(ctxWithTimeout, &proto.IncrementRequest{})
		if err != nil {
			log.Printf("could not increment: %v", err)
			continue
		}
		log.Printf("Current value: %d", resp.GetCurrentValue())
		time.Sleep(1 * time.Second)
	}
}

// Тестовый клиент для grpc
package main

import (
	"context"
	"log"

	pb "github.com/iurnickita/vigilant-train/internal/shortener/grpc_server/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func main() {
	// устанавливаем соединение с сервером
	conn, err := grpc.NewClient(":3200", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	// получаем переменную интерфейсного типа ShortenerClient,
	// через которую будем отправлять сообщения
	c := pb.NewShortenerClient(conn)

	// функция, в которой будем отправлять сообщения
	testUsers(c)
}

func testUsers(c pb.ShortenerClient) {

	ctx := context.Background()
	url := "https://monkeytype.com/"

	log.Print("Получение токена")
	regResp, err := c.Register(ctx, &pb.Empty{})
	if err != nil {
		log.Fatal(err)
	}
	md := metadata.Pairs("token", regResp.Token)
	ctx = metadata.NewOutgoingContext(ctx, md)

	log.Printf("Создается короткая ссылка для URL: %s", url)
	setResp, err := c.SetShortener(ctx, &pb.SetShortenerRequest{Url: url})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Получена короткая сслыка: %s", setResp.Code)

	log.Printf("Поптыка перехода по короткой ссылке: %s", setResp.Code)
	getResp, err := c.GetShortener(ctx, &pb.GetShortenerRequest{Code: setResp.Code})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Получена URL: %s", getResp.Url)

	log.Print("Попытка удалить")
	_, err = c.DeleteShortenerBatch(ctx, &pb.DeleteShortenerBatchRequest{Code: []string{setResp.Code}})
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Успех")

	log.Printf("Поптыка перехода по короткой ссылке: %s", setResp.Code)
	getResp, err = c.GetShortener(ctx, &pb.GetShortenerRequest{Code: setResp.Code})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Получена URL: %s", getResp.Url)
}

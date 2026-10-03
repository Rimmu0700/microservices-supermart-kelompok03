package main

import (
	"context"
	"log"
	"time"

	catalogv1 "github.com/nusantara-supermart/backend/proto/catalog/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func main() {
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("gagal membuat koneksi: %v", err)
	}
	defer conn.Close()
	client := catalogv1.NewCatalogServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 1) Unary: produk yang ada
	r1, err := client.GetProduct(ctx, &catalogv1.GetProductRequest{ProductId: "p0000001-0000-0000-0000-000000000001"})
	if err != nil {
		log.Fatalf("GetProduct gagal: %v", err)
	}
	p := r1.GetProduct()
	log.Printf("[Unary] %s | %s | %s %d | terbit=%t",
		p.GetProductId(), p.GetTitle(), p.GetPrice().GetCurrencyCode(), p.GetPrice().GetAmount(), p.GetIsPublished())

	// 2) Unary: produk yang tidak ada -> NotFound
	_, err = client.GetProduct(ctx, &catalogv1.GetProductRequest{ProductId: "tidak-ada"})
	if st, _ := status.FromError(err); st.Code() == codes.NotFound {
		log.Printf("[Unary] NotFound sesuai harapan: %s", st.Message())
	} else {
		log.Printf("[Unary] respons tak terduga: %v", err)
	}

	// 3) Batch: 3 ID, 1 di antaranya tidak ada
	r3, err := client.BatchGetProducts(ctx, &catalogv1.BatchGetProductsRequest{ProductIds: []string{
		"p0000001-0000-0000-0000-000000000001",
		"p0000001-0000-0000-0000-000000000002",
		"id-tidak-ada",
	}})
	if err != nil {
		log.Fatalf("BatchGetProducts gagal: %v", err)
	}
	log.Printf("[Batch] ditemukan=%d, tidak ditemukan=%v", len(r3.GetProducts()), r3.GetNotFoundIds())
	for _, it := range r3.GetProducts() {
		log.Printf("        - %s | %s | %d", it.GetProductId(), it.GetTitle(), it.GetPrice().GetAmount())
	}
}

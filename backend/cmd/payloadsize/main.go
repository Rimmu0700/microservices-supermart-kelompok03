package main

import (
	"fmt"

	catalogv1 "github.com/nusantara-supermart/backend/proto/catalog/v1"
	"google.golang.org/protobuf/proto"
)

func main() {
	resp := &catalogv1.GetProductResponse{
		Product: &catalogv1.ProductSummary{
			ProductId:   "p0000001-0000-0000-0000-000000000001",
			Title:       "Beras Merah Organik Tabanan 5 Kg",
			Price:       &catalogv1.Money{CurrencyCode: "IDR", Amount: 95000},
			IsPublished: true,
		},
	}
	b, _ := proto.Marshal(resp)
	fmt.Printf("Ukuran Protobuf (body): %d bytes\n", len(b))
}

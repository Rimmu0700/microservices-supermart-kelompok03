package order

import (
	"context"
	"errors"
	"time"

	catalogv1 "github.com/nusantara-supermart/backend/proto/catalog/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// ErrProductNotFound dikembalikan bila Catalog menjawab NotFound.
var ErrProductNotFound = errors.New("product not found")

// ProductInfo adalah ringkasan produk yang diterima Order dari Catalog (kontrak gRPC).
type ProductInfo struct {
	ID          string
	Title       string
	Price       int64 // rupiah
	IsPublished bool
}

type grpcCatalogClient struct {
	client  catalogv1.CatalogServiceClient
	timeout time.Duration
}

// NewGRPCCatalogClient membuat klien gRPC ke catalog-service (alamat mis. "catalog-service:50051").
// grpc.NewClient bersifat lazy: koneksi baru dibuat saat panggilan pertama.
func NewGRPCCatalogClient(addr string) (CatalogClient, func() error, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	return &grpcCatalogClient{
		client:  catalogv1.NewCatalogServiceClient(conn),
		timeout: 2 * time.Second,
	}, conn.Close, nil
}

func (g *grpcCatalogClient) GetProduct(productID string) (*ProductInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), g.timeout)
	defer cancel()

	res, err := g.client.GetProduct(ctx, &catalogv1.GetProductRequest{ProductId: productID})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrProductNotFound
		}
		return nil, err
	}
	p := res.GetProduct()
	return &ProductInfo{
		ID:          p.GetProductId(),
		Title:       p.GetTitle(),
		Price:       p.GetPrice().GetAmount(),
		IsPublished: p.GetIsPublished(),
	}, nil
}

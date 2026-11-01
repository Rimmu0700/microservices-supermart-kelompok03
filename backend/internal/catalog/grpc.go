package catalog

import (
	"context"
	"log"
	"math"
	"net"

	catalogv1 "github.com/nusantara-supermart/backend/proto/catalog/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const maxBatchSize = 100

type GRPCServer struct {
	catalogv1.UnimplementedCatalogServiceServer
	svc Service
}

func NewGRPCServer(svc Service) *GRPCServer { return &GRPCServer{svc: svc} }

// Harga disimpan float64 di database/model; di kontrak dikirim sebagai integer rupiah.
func toProto(p ProductSummary) *catalogv1.ProductSummary {
	return &catalogv1.ProductSummary{
		ProductId:   p.ID,
		Title:       p.Title,
		Price:       &catalogv1.Money{CurrencyCode: "IDR", Amount: int64(math.Round(p.BasePrice))},
		IsPublished: p.IsPublished,
	}
}

func (s *GRPCServer) GetProduct(ctx context.Context, req *catalogv1.GetProductRequest) (*catalogv1.GetProductResponse, error) {
	if req.GetProductId() == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id wajib diisi")
	}
	list, err := s.svc.GetProductSummaries([]string{req.GetProductId()})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "gagal membaca produk: %v", err)
	}
	if len(list) == 0 {
		return nil, status.Errorf(codes.NotFound, "produk %s tidak ditemukan", req.GetProductId())
	}
	return &catalogv1.GetProductResponse{Product: toProto(list[0])}, nil
}

func (s *GRPCServer) BatchGetProducts(ctx context.Context, req *catalogv1.BatchGetProductsRequest) (*catalogv1.BatchGetProductsResponse, error) {
	ids := req.GetProductIds()
	if len(ids) == 0 {
		return nil, status.Error(codes.InvalidArgument, "product_ids tidak boleh kosong")
	}
	if len(ids) > maxBatchSize {
		return nil, status.Errorf(codes.InvalidArgument, "maksimal %d produk per permintaan", maxBatchSize)
	}

	list, err := s.svc.GetProductSummaries(ids)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "gagal membaca produk: %v", err)
	}

	found := make(map[string]bool, len(list))
	resp := &catalogv1.BatchGetProductsResponse{}
	for _, p := range list {
		found[p.ID] = true
		resp.Products = append(resp.Products, toProto(p))
	}
	for _, id := range ids {
		if !found[id] {
			resp.NotFoundIds = append(resp.NotFoundIds, id)
		}
	}
	return resp, nil
}

// ServeGRPC membuka listener gRPC (HTTP/2), mis. addr = ":50051".
func ServeGRPC(addr string, svc Service) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	g := grpc.NewServer()
	catalogv1.RegisterCatalogServiceServer(g, NewGRPCServer(svc))
	log.Printf("Catalog gRPC server listening on %s", addr)
	return g.Serve(lis)
}

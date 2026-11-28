// Package internalclient berisi klien HTTP untuk panggilan internal antar service
// (perintah yang dulu berupa pemanggilan fungsi langsung di dalam monolit).
package internalclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nusantara-supermart/backend/pkg/replica"
)

type base struct {
	baseURL string
	http    *http.Client
}

func newBase(baseURL string) base {
	return base{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

func (b base) do(method, path string, body interface{}) error {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, b.baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(replica.TokenHeader, replica.Token())

	resp, err := b.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		return fmt.Errorf("%s %s -> status %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	return nil
}

// OrderClient memanggil order-service. Memenuhi interface OrderService milik payment dan logistics.
type OrderClient struct{ b base }

func NewOrderClient(baseURL string) *OrderClient { return &OrderClient{b: newBase(baseURL)} }

func (c *OrderClient) UpdateStatus(orderID, statusCode, notes string) error {
	return c.b.do(http.MethodPut, "/internal/orders/"+url.PathEscape(orderID)+"/status",
		map[string]string{"status": statusCode, "notes": notes})
}

// InventoryClient memanggil inventory-service. Memenuhi interface StockClient milik order.
type InventoryClient struct{ b base }

func NewInventoryClient(baseURL string) *InventoryClient {
	return &InventoryClient{b: newBase(baseURL)}
}

func (c *InventoryClient) ReleaseStock(orderID string) error {
	return c.b.do(http.MethodPost, "/internal/stock/release/"+url.PathEscape(orderID), nil)
}

// Pindah/pilih database catalog
db = db.getSiblingDB('catalog_service_db');

// Buat koleksi untuk kategori produk
db.createCollection('categories');

// Buat koleksi untuk produk
db.createCollection('products');

// Index supaya pencarian nama produk lebih cepat
db.products.createIndex({ name: 1 });

// Contoh data awal (seed)
db.categories.insertOne({
  name: "Makanan & Minuman",
  description: "Kategori produk makanan dan minuman"
});

db.products.insertOne({
  name: "Contoh Produk",
  category: "Makanan & Minuman",
  price: 15000,
  stock_ref: null
});
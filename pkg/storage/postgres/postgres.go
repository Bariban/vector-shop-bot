package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	_ "github.com/lib/pq"

	"github.com/Bariban/vector-shop-bot/pkg/storage"

	"github.com/shopspring/decimal"
)

type Storage struct {
	db *sql.DB
}

// New создает новое подключение к PostgreSQL.
func New() (*Storage, error) {
	connStr := "postgresql://postgres:admin@localhost:5432/vectorshop_db?sslmode=disable" // Укажите правильные креды
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("can't open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("can't connect to database: %w", err)
	}

	return &Storage{db: db}, nil
}

// Save сохраняет продукт в базе данных.
func (s *Storage) Save(ctx context.Context, p *storage.Product) (uint, error) {
	q := `INSERT INTO Products (user_name, name, description, count, purchase_price, selling_price) 
		  VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`

	var ID uint
	err := s.db.QueryRowContext(ctx, q, p.UserName, p.Name, p.Description, p.Count, p.PurchasePrice.String(), p.SellingPrice.String()).Scan(&ID)
	if err != nil {
		return 0, fmt.Errorf("can't save product: %w", err)
	}

	return ID, nil
}

// AddOrder сохраняет заказ
func (s *Storage) AddOrder(ctx context.Context, o *storage.Order) (uint, error) {
	q := `INSERT INTO Orders (username, amount, date, pay_type_id, buyers_phone) 
		  VALUES ($1, $2, $3, $4, $5) RETURNING id`
	var ID uint
	err := s.db.QueryRowContext(ctx, q, o.UserName, o.Amount, o.Date, o.PayType.ID, o.BuersPhone).Scan(&ID)
	if err != nil {
		return 0, fmt.Errorf("can't save order: %w", err)
	}

	return ID, nil
}

// AddOrderDetail сохраняет детали заказа
func (s *Storage) AddOrderDetail(ctx context.Context, o *storage.OrderDetail) (uint, error) {
	q := `INSERT INTO Orders (order_id, product_id, amount, count, discount, factSum) 
		  VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
	var ID uint
	err := s.db.QueryRowContext(ctx, q, o.OrderID, o.ProductID, o.Amount, o.Count, o.Discount, o.FactSum).Scan(&ID)
	if err != nil {
		return 0, fmt.Errorf("can't save order details: %w", err)
	}

	return ID, nil
}

// UpdateProductField обновляет параметр товара
func (s *Storage) UpdateProductField(ctx context.Context, productID uint, field string, value interface{}) error {
	query := fmt.Sprintf("UPDATE products SET %s = $1 WHERE id = $2", field)
	_, err := s.db.ExecContext(ctx, query, value, productID)
	if err != nil {
		return fmt.Errorf("ошибка обновления поля %s: %w", field, err)
	}
	return nil
}

// SaveImage добавляет изображение в таблицу Images, привязывая его к товару по product_id.
func (s *Storage) SaveImage(ctx context.Context, p *storage.Product) error {
	q := `INSERT INTO Images (product_id, username, blob_content, vector) VALUES ($1, $2, $3, $4)`

	for _, image := range p.Image {
		_, err := s.db.ExecContext(ctx, q, p.ProductID, p.UserName, image.Byte, Float64SliceToString(image.Float))
		if err != nil {
			return fmt.Errorf("can't save photo: %w", err)
		}
	}

	return nil
}

// GetPhotosByProductID возвращает список байтовых массивов (контентов фото) для указанного productID.
func (s *Storage) GetPhotosByProductID(ctx context.Context, productID uint) ([][]byte, error) {
	q := `SELECT blob_content FROM Images WHERE product_id = $1`

	rows, err := s.db.QueryContext(ctx, q, productID)
	if err != nil {
		log.Printf("can't get photos for product: %v", err)
		return nil, fmt.Errorf("can't get photos for product: %w", err)
	}
	defer rows.Close()

	var photos [][]byte
	for rows.Next() {
		var content []byte
		if err := rows.Scan(&content); err != nil {
			return nil, fmt.Errorf("can't scan photo content: %w", err)
		}
		photos = append(photos, content)
	}

	return photos, nil
}

// GetVectorsByUsername возвращает список числовых массивов (контентов фото) для указанного username.
func (s *Storage) GetVectorsByUsername(ctx context.Context, username string) ([]*storage.ImageMeta, error) {
	q := `SELECT product_id, blob_content, vector FROM Images WHERE username = $1`

	rows, err := s.db.QueryContext(ctx, q, username)
	if err != nil {
		log.Printf("can't get photos for product: %v", err)
		return nil, fmt.Errorf("can't get photos for product: %w", err)
	}
	defer rows.Close()

	var images []*storage.ImageMeta
	for rows.Next() {
		var vectorStr string
		var productID uint
		var byte []byte
		if err := rows.Scan(&productID, &byte, &vectorStr); err != nil {
			return nil, fmt.Errorf("can't scan photo content: %w", err)
		}

		vector, err := StringToFloat64Slice(vectorStr)
		if err != nil {
			return nil, fmt.Errorf("can't convert vector string to slice: %w", err)
		}

		// Создаем объект ImageMeta и добавляем его в срез
		imageMeta := &storage.ImageMeta{
			ProductID: productID,
			Byte:      byte,
			Float:     vector,
		}
		images = append(images, imageMeta)
	}

	// Проверяем ошибки после цикла rows.Next()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return images, nil
}

// GetProducts возвращает список продуктов по имени пользователя.
func (s *Storage) GetProducts(ctx context.Context, userName string) ([]*storage.Product, error) {
	q := `SELECT id, user_name, name, description, count, purchase_price, selling_price 
	      FROM Products WHERE user_name = $1`

	rows, err := s.db.QueryContext(ctx, q, userName)
	if err != nil {
		return nil, fmt.Errorf("can't get products by user: %w", err)
	}
	defer rows.Close()

	var products []*storage.Product
	for rows.Next() {
		var p storage.Product
		var purchasePrice, sellingPrice string

		err := rows.Scan(
			&p.ProductID, &p.UserName, &p.Name, &p.Description, &p.Count, &purchasePrice, &sellingPrice,
		)
		if err != nil {
			return nil, fmt.Errorf("can't scan product row: %w", err)
		}

		p.PurchasePrice, _ = decimal.NewFromString(purchasePrice)
		p.SellingPrice, _ = decimal.NewFromString(sellingPrice)

		products = append(products, &p)
	}

	return products, nil
}

func (s *Storage) GetProductByID(ctx context.Context, productID uint) (*storage.Product, error) {
	query := `SELECT id, user_name, name, description, count, purchase_price, selling_price 
              FROM products WHERE id = $1`
	row := s.db.QueryRowContext(ctx, query, productID)

	product := &storage.Product{}
	err := row.Scan(
		&product.ProductID,
		&product.UserName,
		&product.Name,
		&product.Description,
		&product.Count,
		&product.PurchasePrice,
		&product.SellingPrice,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Товар не найден
		}
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}

	return product, nil
}

// Remove удаляет продукт из базы данных.
func (s *Storage) Remove(ctx context.Context, productID uint) error {
	q := `DELETE FROM Products WHERE id = $1`

	_, err := s.db.ExecContext(ctx, q, productID)
	if err != nil {
		return fmt.Errorf("can't remove product: %w", err)
	}
	q = `DELETE FROM Images WHERE product_id = $1`

	_, err = s.db.ExecContext(ctx, q, productID)
	if err != nil {
		return fmt.Errorf("can't remove product images: %w", err)
	}

	return nil
}

// IsExistsVector проверяет, существует ли продукт в базе данных по `id`.
func (s *Storage) IsExistsVector(ctx context.Context, productID uint) (bool, error) {
	q := `SELECT COUNT(*) FROM Products WHERE id = $1`

	var count int
	err := s.db.QueryRowContext(ctx, q, productID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("can't check if product exists: %w", err)
	}

	return count > 0, nil
}

// UpdProduct обновляет параметры товара
func (s *Storage) UpdProduct(ctx context.Context, productID uint, param string, value string) error {
	q := fmt.Sprintf(`UPDATE Products SET %s = $1 WHERE id = $2`, param)

	_, err := s.db.ExecContext(ctx, q, value, productID)
	if err != nil {
		return fmt.Errorf("can't update product: %w", err)
	}

	return nil
}

// Init создает таблицы Products и Images, если их еще нет.
func (s *Storage) Init(ctx context.Context) error {
	q1 := `CREATE TABLE IF NOT EXISTS products (
		id SERIAL PRIMARY KEY,
		user_name TEXT,
		name TEXT,
		description TEXT,
		count INTEGER,
		purchase_price TEXT,
		selling_price TEXT
	)`

	_, err := s.db.ExecContext(ctx, q1)
	if err != nil {
		return fmt.Errorf("can't create Products table: %w", err)
	}

	q2 := `CREATE TABLE IF NOT EXISTS images (
		id SERIAL PRIMARY KEY,
		username TEXT,
		product_id SERIAL,
		blob_content BYTEA,
		vector TEXT
		)`

	_, err = s.db.ExecContext(ctx, q2)
	if err != nil {
		return fmt.Errorf("can't create Images table: %w", err)
	}
	

	q3 := `CREATE TABLE IF NOT EXISTS orders (
		id SERIAL PRIMARY KEY,
		username TEXT,
		amount NUMERIC(10, 2),
		date DATE,
		pay_type_id NUMERIC(2),
		buyers_phone TEXT
		)`

	_, err = s.db.ExecContext(ctx, q3)
	if err != nil {
		return fmt.Errorf("can't create Sales table: %w", err)
	}

	q4 := `CREATE TABLE IF NOT EXISTS order_details (
		id SERIAL PRIMARY KEY,
		sale_id NUMERIC NOT NULL,
		product_id NUMERIC NOT NULL,
		amount NUMERIC(10, 2) NOT NULL,
		count NUMERIC(10) NOT NULL,
		discount NUMERIC(3),
		factSum NUMERIC(10, 2) NOT NULL
	);`

	_, err = s.db.ExecContext(ctx, q4)
	if err != nil {
		return fmt.Errorf("can't create SalesDetails table: %w", err)
	}
	
	q5 := `CREATE TABLE IF NOT EXISTS pay_types (
		id SERIAL PRIMARY KEY,
		description text
	);`

	_, err = s.db.ExecContext(ctx, q5)
	if err != nil {
		return fmt.Errorf("can't create SalesDetails table: %w", err)
	}
	return nil
}

// Конвертация []float64 в строку
func Float64SliceToString(slice []float64) string {
	// Преобразуем каждый элемент в строку и соединяем через запятую
	strSlice := make([]string, len(slice))
	for i, num := range slice {
		strSlice[i] = strconv.FormatFloat(num, 'f', -1, 64) // Без ограничения точности
	}
	return strings.Join(strSlice, ",")
}

// Конвертация строки обратно в []float64
func StringToFloat64Slice(str string) ([]float64, error) {
	// Разделяем строку по запятой
	strSlice := strings.Split(str, ",")
	floatSlice := make([]float64, len(strSlice))

	// Конвертируем каждый элемент в float64
	for i, s := range strSlice {
		num, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, fmt.Errorf("не удалось преобразовать '%s' в float64: %w", s, err)
		}
		floatSlice[i] = num
	}
	return floatSlice, nil
}

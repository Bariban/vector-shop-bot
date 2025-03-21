package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

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
	q := `INSERT INTO Products (user_id, name, description, count, purchase_price, selling_price, shop_id, bar_code) 
		  VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`

	var ID uint
	err := s.db.QueryRowContext(ctx, q,
		p.UserID,
		p.Name,
		p.Description,
		p.Count,
		p.PurchasePrice.String(),
		p.SellingPrice.String(),
		p.ShopID,
		p.BarCode).Scan(&ID)
	if err != nil {
		return 0, fmt.Errorf("can't save product: %w", err)
	}

	return ID, nil
}

// AddOrderWithDetails сохраняет заказ и детали в одной транзакции
func (s *Storage) AddOrderWithDetails(ctx context.Context, order *storage.Order) (uint, error) {
	// Начинаем транзакцию
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("не удалось начать транзакцию: %w", err)
	}

	// Вставляем заказ
	orderID := uint(0)
	queryOrder := `INSERT INTO Orders (user_id, amount, pay_type_id, buyers_phone, shop_id) 
                   VALUES ($1, $2, $3, $4, $5) RETURNING id`
	err = tx.QueryRowContext(ctx, queryOrder, order.UserID, order.Amount, order.PayType.ID, order.BuersPhone, order.ShopID).Scan(&orderID)
	if err != nil {
		tx.Rollback() // Откат транзакции
		return 0, fmt.Errorf("не удалось сохранить заказ: %w", err)
	}

	// Вставляем детали заказа
	queryDetail := `INSERT INTO Order_Details (order_id, product_id, amount, count, discount, fact_sum) 
                    VALUES ($1, $2, $3, $4, $5, $6)`
	for _, detail := range order.Details {
		_, err = tx.ExecContext(ctx, queryDetail, orderID, detail.ProductID, detail.Amount, detail.Count, detail.Discount, detail.FactSum)
		if err != nil {
			tx.Rollback() // Откат транзакции
			return 0, fmt.Errorf("не удалось сохранить детали заказа: %w", err)
		}
		// Обновляем количество товара
		queryUpdateProduct := `UPDATE products SET count = count - $1 WHERE id = $2 AND count >= $1`
		res, err := tx.ExecContext(ctx, queryUpdateProduct, detail.Count, detail.ProductID)
		if err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("не удалось обновить количество товара %d: %w", detail.ProductID, err)
		}

		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 0 {
			tx.Rollback()
			return 0, fmt.Errorf("недостаточно товара")
		}
	}

	// Завершаем транзакцию
	err = tx.Commit()
	if err != nil {
		return 0, fmt.Errorf("не удалось завершить транзакцию: %w", err)
	}

	return orderID, nil
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

// UpdateShopField обновляет параметр магазина
func (s *Storage) UpdateShopField(ctx context.Context, shopID uint, field string, value interface{}) error {
	query := fmt.Sprintf("UPDATE shops SET %s = $1 WHERE id = $2", field)
	_, err := s.db.ExecContext(ctx, query, value, shopID)
	if err != nil {
		return fmt.Errorf("ошибка обновления поля %s: %w", field, err)
	}
	return nil
}

// SaveImage добавляет изображение в таблицу Images, привязывая его к товару по product_id.
func (s *Storage) SaveImage(ctx context.Context, p *storage.Product) (*storage.Product, error) {
	q := `INSERT INTO Images (product_id, user_id, blob_content, vector, shop_id) 
          VALUES ($1, $2, $3, $4, $5) RETURNING id`

	for i, image := range p.Image {
		if image.Float == nil {
			continue
		}
		var imageID uint
		err := s.db.QueryRowContext(ctx, q,
			p.ProductID,
			p.UserID,
			image.Byte,
			image.Float,
			p.ShopID).Scan(&imageID)
		if err != nil {
			return nil, fmt.Errorf("can't save photo: %w", err)
		}
		p.Image[i].ImageID = imageID
	}

	return p, nil
}

// GetPhotosByProductID возвращает список байтовых массивов (контентов фото) для указанного productID.
func (s *Storage) GetPhotosByProductID(ctx context.Context, productID uint) ([]storage.ImageMeta, error) {
	q := `SELECT id, blob_content  FROM Images WHERE product_id = $1`

	rows, err := s.db.QueryContext(ctx, q, productID)
	if err != nil {
		log.Printf("can't get photos for product: %v", err)
		return nil, fmt.Errorf("can't get photos for product: %w", err)
	}
	defer rows.Close()
	var images []storage.ImageMeta
	for rows.Next() {
		var image storage.ImageMeta
		if err := rows.Scan(&image.ImageID, &image.Byte); err != nil {
			return nil, fmt.Errorf("can't scan photo content: %w", err)
		}
		images = append(images, image)
	}

	return images, nil
}

// GetPhotosByImageID возвращает список байтовых массивов (контентов фото) для указанного ImageID.
func (s *Storage) GetVectorByImageID(ctx context.Context, ImageID uint) ([]float32, error) {
	q := `SELECT vector FROM Images WHERE id = $1`

	rows, err := s.db.QueryContext(ctx, q, ImageID)
	if err != nil {
		log.Printf("can't get photos for product: %v", err)
		return nil, fmt.Errorf("can't get photos for product: %w", err)
	}
	defer rows.Close()

	var vector []float32
	for rows.Next() {
		if err := rows.Scan(&vector); err != nil {
			return nil, fmt.Errorf("can't scan photo content: %w", err)
		}
	}

	return vector, nil
}

// GetProducts возвращает список продуктов по имени пользователя.
func (s *Storage) GetProducts(ctx context.Context, shopID uint) ([]*storage.Product, error) {
	q := `SELECT id, user_name, name, description, count, purchase_price, selling_price 
	      FROM Products WHERE shop_id = $1`

	rows, err := s.db.QueryContext(ctx, q, shopID)
	if err != nil {
		return nil, fmt.Errorf("can't get products by user: %w", err)
	}
	defer rows.Close()

	var products []*storage.Product
	for rows.Next() {
		var p storage.Product
		var purchasePrice, sellingPrice string

		err := rows.Scan(
			&p.ProductID, &p.UserID, &p.Name, &p.Description, &p.Count, &purchasePrice, &sellingPrice,
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

// GetNearestNeighbors возвращает ближайшие вектора (по косинусной близости) к заданному вектору.
func (s *Storage) GetNearestNeighbors(ctx context.Context, images []*storage.ImageMeta, limit int) ([]*storage.Product, error) {
	var products []*storage.Product
	for _, inputImage := range images {
		// Проверяем, что вектор не пустой
		if inputImage.Float == nil {
			continue
		}
		q := `SELECT id, user_name, name, description, count, purchase_price, selling_price 
		  FROM Products WHERE shop_id = $1
		  ORDER BY vector <-> $1
		  LIMIT $2
		`

		rows, err := s.db.QueryContext(ctx, q, inputImage.Float, limit)
		if err != nil {
			log.Printf("can't find nearest neighbors: %v", err)
			return nil, fmt.Errorf("can't find nearest neighbors: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var p storage.Product
			var purchasePrice, sellingPrice string

			err := rows.Scan(
				&p.ProductID, &p.UserID, &p.Name, &p.Description, &p.Count, &purchasePrice, &sellingPrice,
			)
			if err != nil {
				return nil, fmt.Errorf("can't scan product row: %w", err)
			}

			p.PurchasePrice, _ = decimal.NewFromString(purchasePrice)
			p.SellingPrice, _ = decimal.NewFromString(sellingPrice)

			products = append(products, &p)
		}
	}

	return products, nil
}

// GetProductsByBarCode возвращает список продуктов по имени пользователя по штрихкоду.
func (s *Storage) GetProductsByBarCode(ctx context.Context, shopID uint, barCode string) ([]*storage.Product, error) {
	q := `SELECT id, name, description, count, purchase_price, selling_price 
	      FROM Products WHERE shop_id = $1 and bar_code = $2`

	rows, err := s.db.QueryContext(ctx, q, shopID, barCode)
	if err != nil {
		return nil, fmt.Errorf("can't get products by user: %w", err)
	}
	defer rows.Close()

	var products []*storage.Product
	for rows.Next() {
		var p storage.Product
		var purchasePrice, sellingPrice string

		err := rows.Scan(
			&p.ProductID, &p.Name, &p.Description, &p.Count, &purchasePrice, &sellingPrice,
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
	query := `SELECT id, user_id, name, description, count, purchase_price, selling_price 
              FROM products WHERE id = $1`
	row := s.db.QueryRowContext(ctx, query, productID)

	product := &storage.Product{}
	err := row.Scan(
		&product.ProductID,
		&product.UserID,
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
func (s *Storage) RemoveProduct(ctx context.Context, productID uint) error {
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

// Remove удаляет продукт из базы данных.
func (s *Storage) DeleteImage(ctx context.Context, imageID uint) error {

	q := `DELETE FROM Images WHERE id = $1`

	_, err := s.db.ExecContext(ctx, q, imageID)
	if err != nil {
		return fmt.Errorf("can't remove image: %w", err)
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

// GetVectorsByUserName возвращает список векторов по имени пользователя.
func (s *Storage) GetVectorsByShopID(ctx context.Context, shopID uint) ([]*storage.ImageMeta, error) {
	q := `SELECT id, vector FROM Images WHERE shop_id = $1`

	rows, err := s.db.QueryContext(ctx, q, shopID)
	if err != nil {
		return nil, fmt.Errorf("can't get vectors by user: %w", err)
	}
	defer rows.Close()

	var imageMeta []*storage.ImageMeta
	for rows.Next() {
		var id uint
		var vector []float32 // Временный массив для чтения из PostgreSQL

		err := rows.Scan(&id, &vector)
		if err != nil {
			return nil, fmt.Errorf("can't scan vector row: %w", err)
		}

		imageMeta = append(imageMeta, &storage.ImageMeta{
			ImageID: id,
			Float:   vector,
		})
	}

	return imageMeta, nil
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
		shop_id SERIAL,
		user_id SERIAL,
		name TEXT,
		description TEXT,
		count INTEGER,
		purchase_price TEXT,
		selling_price TEXT,
		bar_code TEXT
	)`

	_, err := s.db.ExecContext(ctx, q1)
	if err != nil {
		return fmt.Errorf("can't create Products table: %w", err)
	}

	q2 := `CREATE TABLE IF NOT EXISTS images (
		id SERIAL PRIMARY KEY,
		user_id serial,
		shop_id serial,
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
		user_id serial NOT NULL,
		shop_id serial NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
		amount NUMERIC(10, 2) NOT NULL,
		date DATE NOT NULL DEFAULT now(),
		pay_type_id NUMERIC(2),
		buyers_phone TEXT
		)`

	_, err = s.db.ExecContext(ctx, q3)
	if err != nil {
		return fmt.Errorf("can't create orders table: %w", err)
	}

	q4 := `CREATE TABLE IF NOT EXISTS order_details (
		id SERIAL PRIMARY KEY,
		order_id NUMERIC NOT NULL,
		product_id NUMERIC NOT NULL,
		amount NUMERIC(10, 2) NOT NULL,
		count NUMERIC(10) NOT NULL,
		discount NUMERIC(3),
		fact_sum NUMERIC(10, 2) NOT NULL
	);`

	_, err = s.db.ExecContext(ctx, q4)
	if err != nil {
		return fmt.Errorf("can't create order_details table: %w", err)
	}

	q5 := `CREATE TABLE IF NOT EXISTS pay_types (
		id SERIAL PRIMARY KEY,
		description text
	);`

	_, err = s.db.ExecContext(ctx, q5)
	if err != nil {
		return fmt.Errorf("can't create pay_types table: %w", err)
	}

	q6 := `CREATE TABLE IF NOT EXISTS shops (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255),
		description VARCHAR(255),
		owner_user_id VARCHAR(50) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = s.db.ExecContext(ctx, q6)
	if err != nil {
		return fmt.Errorf("can't create shops table: %w", err)
	}

	q7 := `CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		telegram_user_id SERIAL NOT NULL,
		shop_id serial NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
		username VARCHAR(255),
		first_name VARCHAR(30),
		last_name VARCHAR(30),
		role VARCHAR(50) NOT NULL, -- 'admin', 'customer', 'clerk'
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_activity_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE (shop_id, telegram_user_id)
	);`

	_, err = s.db.ExecContext(ctx, q7)
	if err != nil {
		return fmt.Errorf("can't create shop_users table: %w", err)
	}

	return nil
}

func (s *Storage) CreateShop(ctx context.Context, name, userID uint) (int, error) {
	query := `INSERT INTO shops (name, owner_user_id, created_at) VALUES ($1, $2, NOW()) RETURNING id`
	var shopID int
	err := s.db.QueryRowContext(ctx, query, name, userID).Scan(&shopID)
	if err != nil {
		return 0, fmt.Errorf("error creating shop: %w", err)
	}
	return shopID, nil
}

func (s *Storage) GetShopName(ctx context.Context, shopID int) (string, error) {
	query := `SELECT name FROM shops WHERE id = $1`
	var name string
	err := s.db.QueryRowContext(ctx, query, shopID).Scan(&name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil // Пользователь не найден
		}
		return "", fmt.Errorf("error fetching shop name: %w", err)
	}
	return name, nil
}

// GetPhotosByProductID возвращает список байтовых массивов (контентов фото) для указанного productID.
func (s *Storage) GetProductByImageID(ctx context.Context, imageID uint) (*storage.Product, error) {
	query := `SELECT p.*
	FROM products p
	JOIN images i ON i.product_id = p.id
	WHERE i.id = $1
	LIMIT 1
	`

	row := s.db.QueryRowContext(ctx, query, imageID)

	product := &storage.Product{}
	err := row.Scan(
		&product.ProductID,
		&product.ShopID,
		&product.UserID,
		&product.Name,
		&product.Description,
		&product.Count,
		&product.PurchasePrice,
		&product.SellingPrice,
		&product.BarCode,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Товар не найден
		}
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}

	return product, nil
}

// InitUser инициализация пользователя
func (s *Storage) InitUser(user *storage.User) error {
	// Транзакция для проверки и вставки данных
	tx, err := s.db.Begin()
	if err != nil {
		log.Printf("Failed to start transaction: %v", err)
		return err
	}
	defer tx.Rollback()

	var shopID uint
	// Если ShopID = 0, определяем последний магазин, с которым пользователь взаимодействовал
	if user.ShopID == 0 {
		shopID, err = s.getLastShopID(tx, user)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("Failed to get last shop ID: %v", err)
			return err
		}

		// Если магазин не найден, создаем новый
		if err == sql.ErrNoRows {
			shopID, err = s.createShop(tx, user.UserID)
			if err != nil {
				log.Printf("Failed to create shop: %v", err)
				return err
			}
		}
		user.ShopID = shopID
	}

	// Проверяем существование пользователя в конкретном магазине
	exists, err := s.userExists(tx, user.UserID, user.ShopID, user.Role)
	if err != nil {
		log.Printf("Failed to check user existence: %v", err)
		return err
	}

	if exists {
		// Обновляем время последней активности
		err = s.updateUserLastActivity(tx, user.UserID, user.ShopID)
		if err != nil {
			log.Printf("Failed to update user last activity: %v", err)
			return err
		}
	} else {
		// Создаем нового пользователя
		err = s.createUser(tx, user)
		if err != nil {
			log.Printf("Failed to create user: %v", err)
			return err
		}
	}

	// Фиксируем изменения
	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		return err
	}

	return nil
}

// getLastShopID Функция для получения последнего магазина пользователя
func (s *Storage) getLastShopID(tx *sql.Tx, user *storage.User) (uint, error) {
	var shopID uint
	query := `
		SELECT u.shop_id, u.role
		FROM users u
		WHERE u.telegram_user_id = $1
		ORDER BY u.last_activity_at DESC
		LIMIT 1;
	`
	err := tx.QueryRow(query, user.UserID).Scan(&shopID, &user.Role)
	return shopID, err
}

// createShop Функция для создания нового магазина
func (s *Storage) createShop(tx *sql.Tx, userID uint) (uint, error) {
	var shopID uint
	query := `
		INSERT INTO shops (name, description, owner_user_id, created_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING id;
	`
	err := tx.QueryRow(query, "", "", userID).Scan(&shopID)
	return shopID, err
}

// userExists Функция для проверки существования пользователя
func (s *Storage) userExists(tx *sql.Tx, userID uint, shopID uint, role string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM users u
			WHERE u.telegram_user_id = $1 AND u.shop_id = $2 AND u.role = $3
		);
	`
	err := tx.QueryRow(query, userID, shopID, role).Scan(&exists)
	return exists, err
}

// updateUserLastActivity Функция для обновления времени последней активности пользователя
func (s *Storage) updateUserLastActivity(tx *sql.Tx, userID uint, shopID uint) error {
	query := `
		UPDATE users
		SET last_activity_at = NOW()
		WHERE telegram_user_id = $1 AND shop_id = $2;
	`
	_, err := tx.Exec(query, userID, shopID)
	return err
}

// updateUserLastActivity Функция для создания нового пользователя
func (s *Storage) createUser(tx *sql.Tx, user *storage.User) error {
	query := `
		INSERT INTO users (shop_id, telegram_user_id,  username, role, created_at, last_activity_at, first_name, last_name)
		VALUES ($1, $2, $3, $4, NOW(), NOW(), $5, $6);
	`
	_, err := tx.Exec(query, user.ShopID, user.UserID, user.UserName, user.Role, user.FirstName, user.LastName)
	return err
}

// GetUsersByShopID возвращает список пользователей магазина.
func (s *Storage) GetUsersByShopID(ctx context.Context, shopID uint) ([]*storage.User, error) {
	q := `SELECT id, first_name, last_name, username, role 
		  FROM users 
		  WHERE shop_id = $1
		  AND role IN ('customer', 'employee')
		  ORDER BY role`

	rows, err := s.db.QueryContext(ctx, q, shopID)
	if err != nil {
		return nil, fmt.Errorf("can't get users: %w", err)
	}
	defer rows.Close()

	var users []*storage.User
	for rows.Next() {
		var user storage.User
		err := rows.Scan(&user.UserID, &user.FirstName, &user.LastName, &user.UserName, &user.Role)
		if err != nil {
			return nil, fmt.Errorf("can't scan user row: %w", err)
		}

		users = append(users, &user) // Добавляем копию структуры
	}

	// Проверяем ошибки при итерации
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return users, nil
}

// updateUserRole обновление роли пользователя
func (s *Storage) UpdateUserRole(ctx context.Context, userID uint, shopID uint, role string) error {

	query := `
		UPDATE users
		SET role = $3
		WHERE id = $1 AND shop_id = $2;
	`
	_, err := s.db.ExecContext(ctx, query, userID, shopID, role)
	if err != nil {
		return fmt.Errorf("can't update user role: %w", err)
	}

	return nil
}

package storage

import (
	"context"
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

type Storage interface {
	Save(ctx context.Context, p *Product) (uint, error)
	Remove(ctx context.Context, productID uint) error
	IsExists(ctx context.Context, p *Product) (bool, error)
	GetProducts(ctx context.Context, userName string) ([]*Product, error)
	SaveImage(ctx context.Context, p *Product) error
	SearchVector(ctx context.Context, vector []float32) ([]*Product, error)
	GetPhotosByProductID(ctx context.Context, productID uint) ([][]byte, error)
	GetNextProductID(ctx context.Context) (uint, error)
	UpdPhoto(ctx context.Context, p *Product) error
	UpdProduct(ctx context.Context, productID uint, param string, value string) error
	GetProductByID(ctx context.Context, productID uint) (*Product, error)
	AddOrderWithDetails(ctx context.Context, order *Order) (uint, error)
}

var ErrNoSavedProducts = errors.New("no saved Products")

type Product struct {
	ProductID     uint
	ShopID        uint
	UserID        uint
	Name          string
	Description   string
	Count         uint
	PurchasePrice decimal.Decimal
	SellingPrice  decimal.Decimal
	Image         []*ImageMeta
	BarCode       string
}

type ImageMeta struct {
	ImageID   uint
	ProductID uint
	Byte      []byte
	Float     []float32
	Url       string
	BarCode   string
}

type Order struct {
	ID         uint
	UserID     uint
	ShopID     uint
	Amount     decimal.Decimal
	Date       *time.Time
	PayType    *PayType
	Details    []*OrderDetail
	BuersPhone string
}

type PayType struct {
	ID          uint
	Description string
}

type OrderDetail struct {
	ID        uint
	OrderID   uint
	ProductID uint
	Amount    decimal.Decimal
	Count     uint
	Discount  uint
	FactSum   decimal.Decimal
}

type Market struct {
	ID          uint
	Name        string
	Description string
	Locate      string
}

type User struct {
	FirstName string
	LastName  string
	UserName  string
	UserID    uint
	ShopName  string
	ShopID    uint
	Role      string
}

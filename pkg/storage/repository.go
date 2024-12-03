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
	SearchVector(ctx context.Context, vector []float64) ([]*Product, error)
	GetPhotosByProductID(ctx context.Context, productID uint) ([][]byte, error)
	GetNextProductID(ctx context.Context) (uint, error)
	UpdPhoto(ctx context.Context, p *Product) error
	UpdProduct(ctx context.Context, productID uint, param string, value string) error
	GetProductByID(ctx context.Context, productID uint) (*Product, error)
}

var ErrNoSavedProducts = errors.New("no saved Products")

type Product struct {
	ProductID     uint
	UserName      string
	Name          string
	Description   string
	Count         uint
	PurchasePrice decimal.Decimal
	SellingPrice  decimal.Decimal
	Image         []*ImageMeta
}

type ImageMeta struct {
	ImageID   uint
	ProductID uint
	Byte      []byte
	Float     []float64
	Url       string
}

type Order struct {
	ID  uint
	UserName string
	Amount   decimal.Decimal
	Date     *time.Time
	PayType  *PayType
	Details  *[]OrderDetail
	BuersPhone string
}

type PayType struct {
	ID   uint
	Description string
}

type OrderDetail struct {
	ID  uint
	OrderID   uint
	ProductID uint
	Amount    decimal.Decimal
	Count     uint
	Discount  uint
	FactSum   decimal.Decimal
}

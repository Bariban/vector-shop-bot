package recognize

// Модель ответа из clip

type Response struct {
	BarCode  string    `json:"barcode"`
	Features []float32 `json:"features"`
}

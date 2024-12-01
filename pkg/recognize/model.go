package recognize
// Модель ответа из clip

type Recognize interface {
	GetImageVector(fileID string) ([]float64, error)
	ExtractFromModel(imageURL string) ([]float64, error)
}

type Response struct {
	BestCategory  string             `json:"best_category"`
	ExtractedText string             `json:"extracted_text"`
	Features      []float64          `json:"features"`
	Similarities  map[string]float64 `json:"similarities"`
}


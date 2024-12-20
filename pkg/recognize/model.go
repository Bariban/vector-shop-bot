package recognize
// Модель ответа из clip

type Recognize interface {
	GetImageVector(fileID string) ([]float32, error)
	ExtractFromModel(imageURL string) ([]float32, error)
}

type Response struct {
	BestCategory  string             `json:"best_category"`
	ExtractedText string             `json:"extracted_text"`
	Features      []float32          `json:"features"`
	Similarities  map[string]float32 `json:"similarities"`
}


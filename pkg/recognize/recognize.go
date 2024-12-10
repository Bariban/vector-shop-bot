package recognize

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"strings"
)

//EncodeVector принимает массив чисел и возвращает строку в формате base64
func EncodeVector(vector []float64) (string, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(vector); err != nil {
		return "", fmt.Errorf("ошибка кодирования вектора: %w", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

//DecodeVector принимает строку в формате base64 и возвращает массив float32
func DecodeVector(encoded string) ([]float64, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("ошибка декодирования base64: %w", err)
	}

	var vector []float64
	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&vector); err != nil {
		return nil, fmt.Errorf("ошибка декодирования вектора: %w", err)
	}

	return vector, nil
}

// CompareFeatureVectors сравнивает два вектора и возвращает true, если они сходятся.
func CompareFeatureVectors(vector1, vector2 []float64, d float64) (bool, error) {
	// Проверяем, совпадает ли размерность векторов
	if len(vector1) != len(vector2) {
		return false, fmt.Errorf("Vectors have different dimensions: %d vs %d", len(vector1), len(vector2))
	}

	// Вычисляем евклидово расстояние
	var sum float64
	for i := range vector1 {
		diff := vector1[i] - vector2[i]
		sum += diff * diff
	}
	distance := math.Sqrt(sum)

	// Возвращаем true, если расстояние меньше или равно порогу
	return distance <= d, nil
}


//ExtractFromModel извлекает вектор из изображения в URL
func ExtractFromModel(imageURL string) ([]float64, error) {

	// Получаем вектор изображения по URL
	urlClip := "http://127.0.0.1:5000/extract_features"

	// Создание тела запроса
	form := url.Values{}
	form.Add("image_url", imageURL)
	requestBody := strings.NewReader(form.Encode())

	// Создание POST запроса
	req, err := http.NewRequest("POST", urlClip, requestBody)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Отправка запроса
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка отправки запроса: %w", err)
	}
	defer resp.Body.Close()

	// Чтение ответа
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	// Обработка JSON ответа
	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("ошибка разбора JSON ответа: %w", err)
	}

	// Вывод полученных данных
	// fmt.Println("Лучшая категория:", response.BestCategory)
	// fmt.Println("Извлеченный текст:", response.ExtractedText)
	// fmt.Println("Признаки изображения:", response.Features)
	// fmt.Println("Сходство с категориями:", response.Similarities)
	return response.Features, nil
}

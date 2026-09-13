package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCafeNegative(t *testing.T) {
	handler := http.HandlerFunc(mainHandle)

	requests := []struct {
		request string
		status  int
		message string
	}{
		{"/cafe", http.StatusBadRequest, "unknown city"},
		{"/cafe?city=omsk", http.StatusBadRequest, "unknown city"},
		{"/cafe?city=tula&count=na", http.StatusBadRequest, "incorrect count"},
	}
	for _, v := range requests {
		response := httptest.NewRecorder()
		req := httptest.NewRequest("GET", v.request, nil)
		handler.ServeHTTP(response, req)

		assert.Equal(t, v.status, response.Code)
		assert.Equal(t, v.message, strings.TrimSpace(response.Body.String()))
	}
}

func TestCafeWhenOk(t *testing.T) {
	handler := http.HandlerFunc(mainHandle)

	requests := []string{
		"/cafe?count=2&city=moscow",
		"/cafe?city=tula",
		"/cafe?city=moscow&search=ложка",
	}
	for _, v := range requests {
		response := httptest.NewRecorder()
		req := httptest.NewRequest("GET", v, nil)

		handler.ServeHTTP(response, req)

		assert.Equal(t, http.StatusOK, response.Code)
	}
}

func TestCafeCount(t *testing.T) {
	// ШАГ 1: Сначала считаем, сколько всего кафе в Москве.
	// Это нужно, чтобы правильно задать ожидание для count=100.
	totalCafes := len(cafeList["moscow"])
	limit := 100

	// ШАГ 2: Вычисляем, сколько кафе мы ОЖИДАЕМ получить при count=100.
	// Логика: минимум из (100 или всего кафе в городе).
	wantFor100 := limit
	if totalCafes < limit {
		wantFor100 = totalCafes
	}

	// ШАГ 3: Создаем таблицу тестов ТОЛЬКО ОДИН РАЗ здесь, до цикла.
	requests := []struct {
		count int
		want  int
	}{
		{count: 0, want: 0},
		{count: 1, want: 1},
		{count: 2, want: 2},
		// Используем переменную wantFor100, которую мы посчитали выше.
		{count: 100, want: wantFor100},
	}

	// ШАГ 4: Цикл проходит по уже готовой таблице.
	for _, v := range requests {
		params := url.Values{}
		params.Set("city", "moscow")
		params.Set("count", strconv.Itoa(v.count))
		urlStr := "http://localhost:8080/cafe?" + params.Encode()

		resp, err := http.Get(urlStr)
		if err != nil {
			t.Errorf("Ошибка при выполнении запроса: %v", err)
			continue
		}
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Сервер должен вернуть статус 200 OK")

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Errorf("Ошибка чтения тела ответа: %v", err)
			continue
		}
		body := strings.TrimSpace(string(bodyBytes))

		var cafes []string
		if body == "" {
			cafes = []string{}
		} else {
			cafes = strings.Split(body, ",")
		}

		assert.Equal(t, v.want, len(cafes),
			"Для count=%d ожидалось %d кафе, а получено %d",
			v.count, v.want, len(cafes))

	}
}

func TestCafeSearch(t *testing.T) {
	// ШАГ 1: Подготовка тестовой таблицы
	// Здесь мы задаем, что будем искать и сколько результатов ожидаем получить.
	requests := []struct {
		search    string // подстрока для поиска
		wantCount int    // ожидаемое количество найденных кафе
	}{
		{search: "фасоль", wantCount: 0},
		{search: "кофе", wantCount: 2},
		{search: "вилка", wantCount: 1},
	}

	// ШАГ 2: Цикл по каждому варианту поиска
	for _, v := range requests {
		// Формируем параметры запроса
		params := url.Values{}
		params.Set("city", "moscow")
		params.Set("search", v.search) // Передаем искомую подстроку

		urlStr := "http://localhost:8080/cafe?" + params.Encode()

		// Отправляем реальный HTTP-запрос
		resp, err := http.Get(urlStr)
		if err != nil {
			t.Errorf("Ошибка при выполнении запроса: %v", err)
			continue
		}
		defer resp.Body.Close()

		// Проверяем, что сервер ответил успешно
		require.Equal(t, http.StatusOK, resp.StatusCode, "Сервер должен вернуть статус 200 OK")

		// Читаем и чистим тело ответа
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Errorf("Ошибка чтения тела ответа: %v", err)
			continue
		}
		body := strings.TrimSpace(string(bodyBytes))

		var cafes []string
		if body == "" {
			cafes = []string{}
		} else {
			cafes = strings.Split(body, ",")
		}

		// Сначала проверяем, совпадает ли общее число найденных кафе с ожидаемым
		assert.Equal(t, v.wantCount, len(cafes),
			"Для search='%s' ожидалось %d кафе, а получено %d",
			v.search, v.wantCount, len(cafes))

		// И помним про регистр: поиск без учета регистра.

		searchLower := strings.ToLower(v.search)

		for i, cafeName := range cafes {
			// Приводим название кафе к нижнему регистру для сравнения
			cafeNameLower := strings.ToLower(cafeName)

			ok := strings.Contains(cafeNameLower, searchLower)

			assert.True(t, ok,
				"Кафе №%d (%q) не содержит подстроку '%s'",
				i+1, cafeName, v.search)
		}
	}
}

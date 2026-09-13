package main

import (
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

	totalCafes := len(cafeList["moscow"])
	limit := 100
	wantFor100 := limit
	if totalCafes < limit {
		wantFor100 = totalCafes
	}

	requests := []struct {
		count int
		want  int
	}{
		{count: 0, want: 0},
		{count: 1, want: 1},
		{count: 2, want: 2},
		{count: 100, want: wantFor100},
	}

	handler := http.HandlerFunc(mainHandle)

	for _, v := range requests {

		params := url.Values{}
		params.Set("city", "moscow")
		params.Set("count", strconv.Itoa(v.count))

		req := httptest.NewRequest("GET", "/cafe?"+params.Encode(), nil)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		require.Equal(t, http.StatusOK, recorder.Code, "Сервер должен вернуть статус 200 OK")

		body := strings.TrimSpace(recorder.Body.String())

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

	requests := []struct {
		search    string
		wantCount int
	}{
		{search: "фасоль", wantCount: 0},
		{search: "кофе", wantCount: 2},
		{search: "вилка", wantCount: 1},
	}

	handler := http.HandlerFunc(mainHandle)

	for _, v := range requests {

		params := url.Values{}
		params.Set("city", "moscow")
		params.Set("search", v.search)

		req := httptest.NewRequest("GET", "/cafe?"+params.Encode(), nil)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		require.Equal(t, http.StatusOK, recorder.Code, "Сервер должен вернуть статус 200 OK")

		body := strings.TrimSpace(recorder.Body.String())

		var cafes []string
		if body == "" {
			cafes = []string{}
		} else {
			cafes = strings.Split(body, ",")
		}

		// Проверяем количество найденных кафе
		assert.Equal(t, v.wantCount, len(cafes),
			"Для search='%s' ожидалось %d кафе, а получено %d",
			v.search, v.wantCount, len(cafes))

		// Дополнительная проверка: каждое найденное кафе действительно содержит подстроку
		searchLower := strings.ToLower(v.search)
		for i, cafeName := range cafes {
			cafeNameLower := strings.ToLower(cafeName)
			ok := strings.Contains(cafeNameLower, searchLower)
			assert.True(t, ok,
				"Кафе №%d (%q) не содержит подстроку '%s'",
				i+1, cafeName, v.search)
		}
	}
}

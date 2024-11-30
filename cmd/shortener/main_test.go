package main

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-resty/resty/v2"
)

func TestShortener(t *testing.T) {
	testCases := []struct {
		name string
		url  string
	}{
		{
			name: "test #1",
			url:  "https://practicum.yandex.ru/",
		}, {
			name: "test #2",
			url:  "https://ya.ru/",
		},
	}

	// Запуск сервера
	// * непонятно как в такой ситуации подставить httptest.NewServer
	go main()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			// запрос короткой ссылки
			setreq := resty.New().R()
			setreq.Method = http.MethodPost
			setreq.URL = "http://localhost:8080"
			setreq.Body = tc.url
			setresp, err := setreq.Send()
			require.NoError(t, err, "Ошибка отправки запроса Post")

			// обработка ответа
			require.Equal(t, http.StatusCreated, setresp.StatusCode())

			// переход по короткой ссылке
			getrequrl := string(setresp.Body())

			// resty
			// тупит по непонятной причине. Возвращает пустой 200
			/* 			getreq := resty.New().R()
			   			getresp, err := getreq.Get(getrequrl)
			   			require.NoError(t, err, "Ошибка отправки запроса Get")
			   			// обработка ответа * ничего не понятно . Точка в хендлере GetShortener срабатывает, возвращает 301. Тут получаю 200
			   			statuscode := getresp.StatusCode()
			   			location := getresp.Header().Values("Location")*/

			// net/http
			// а тут то же самое
			getresp, err := http.Get(getrequrl)
			require.NoError(t, err, "Ошибка отправки запроса Get")
			statuscode := getresp.StatusCode
			location := getresp.Header.Values("Location")[0]

			require.Equal(t, http.StatusTemporaryRedirect, statuscode)
			require.Equal(t, tc.url, location[0])
		})
	}
}

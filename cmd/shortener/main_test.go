package main

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-resty/resty/v2"
)

func TestShortener(t *testing.T) {
	const DefaultServerAddr = "http://localhost:8080"

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
			setreq.URL = DefaultServerAddr
			setreq.Body = tc.url
			setresp, err := setreq.Send()
			require.NoError(t, err, "Ошибка отправки запроса Post")

			// обработка ответа
			require.Equal(t, http.StatusCreated, setresp.StatusCode())

			// переход по короткой ссылке
			getreq := resty.New().R()
			getresp, err := getreq.Get(string(setresp.Body()))
			require.NoError(t, err, "Ошибка отправки запроса Get")

			// обработка ответа
			require.Equal(t, http.StatusOK, getresp.StatusCode())
			resulturl := getresp.RawResponse.Request.URL.Scheme +
				"://" + getresp.RawResponse.Request.URL.Host + "/"
			require.Equal(t, tc.url, resulturl)
		})
	}
}

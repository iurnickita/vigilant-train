package main

import (
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-resty/resty/v2"

	netutils "github.com/iurnickita/vigilant-train/internal/common/net_utils"
)

func TestShortener(t *testing.T) {
	freePort, err := netutils.GetFreePort()
	require.NoError(t, err, "Ошибка получения свободного порта")
	var ServerAddr = "http://localhost:" + strconv.Itoa(freePort)

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
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"", "-a", ServerAddr, "-b", ServerAddr}
	go main()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			// запрос короткой ссылки
			setreq := resty.New().R()
			setreq.Method = http.MethodPost
			setreq.URL = ServerAddr
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

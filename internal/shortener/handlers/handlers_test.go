package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	handlersConfig "github.com/iurnickita/vigilant-train/internal/shortener/handlers/config"
	"github.com/iurnickita/vigilant-train/internal/shortener/repository"
	repositoryConfig "github.com/iurnickita/vigilant-train/internal/shortener/repository/config"
	"github.com/iurnickita/vigilant-train/internal/shortener/service"
	"go.uber.org/zap"

	"github.com/stretchr/testify/require"
)

func TestHandlers(t *testing.T) {
	tests := []struct {
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

	store, _ := repository.NewStore(repositoryConfig.Config{StoreType: repositoryConfig.StoreTypeVar})
	shortenerService := service.NewShortener(store)
	cfg := handlersConfig.Config{BaseAddr: "localhost:8080"}
	h := newHandlers(cfg, shortenerService, zap.NewNop())

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			// запрос короткой ссылки
			setbody := strings.NewReader(test.url)
			setr := httptest.NewRequest(http.MethodPost, "/", setbody)
			setw := httptest.NewRecorder()
			h.SetShortener(setw, setr)

			// обработка ответа
			setresult := setw.Result()
			require.Equal(t, http.StatusCreated, setresult.StatusCode)
			setresultbody, err := io.ReadAll(setresult.Body)
			require.NoError(t, err)
			err = setresult.Body.Close()
			require.NoError(t, err)

			// переход по короткой ссылке
			lastslashidx := strings.LastIndexByte(string(setresultbody), '/')
			gettarget := string(setresultbody[lastslashidx+1:])
			getr := httptest.NewRequest(http.MethodGet, "/", nil)
			getr.SetPathValue("code", gettarget)
			getw := httptest.NewRecorder()
			h.GetShortener(getw, getr)

			// обработка ответа
			getresult := getw.Result()
			require.Equal(t, http.StatusTemporaryRedirect, getresult.StatusCode)
			getresultlocation := getresult.Header.Values("Location")
			require.Equal(t, getresultlocation[0], test.url)
			err = getresult.Body.Close()
			require.NoError(t, err)
		})
	}

}

func BenchmarkHandlers(b *testing.B) {
	tests := []struct {
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

	store, _ := repository.NewStore(repositoryConfig.Config{StoreType: repositoryConfig.StoreTypeVar})
	shortenerService := service.NewShortener(store)
	cfg := handlersConfig.Config{BaseAddr: "localhost:8080"}
	h := newHandlers(cfg, shortenerService, zap.NewNop())

	// Сброс таймера
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, test := range tests {

			// запрос короткой ссылки
			setbody := strings.NewReader(test.url)
			setr := httptest.NewRequest(http.MethodPost, "/", setbody)
			setw := httptest.NewRecorder()
			h.SetShortener(setw, setr)

			// обработка ответа
			setresult := setw.Result()
			setresultbody, _ := io.ReadAll(setresult.Body)
			setresult.Body.Close()

			// переход по короткой ссылке
			lastslashidx := strings.LastIndexByte(string(setresultbody), '/')
			gettarget := string(setresultbody[lastslashidx+1:])
			getr := httptest.NewRequest(http.MethodGet, "/", nil)
			getr.SetPathValue("code", gettarget)
			getw := httptest.NewRecorder()
			h.GetShortener(getw, getr)
		}
	}
}

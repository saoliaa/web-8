package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"

	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "postgres"
	dbname   = "sandbox"
)

type Handlers struct {
	dbProvider DatabaseProvider
}

type DatabaseProvider struct {
	db *sql.DB
}

// Методы для работы с базой данных
func (dp *DatabaseProvider) SelectQuery() (int, error) {
	var c int
	row := dp.db.QueryRow("SELECT c FROM counter LIMIT 1")
	err := row.Scan(&c)
	if err != nil {
		return 0, err
	}

	return c, nil
}
func (dp *DatabaseProvider) InsertQuery(c int) error {
	_, err := dp.db.Exec("INSERT INTO counter (c) VALUES ($1)", c)
	if err != nil {
		return err
	}

	return nil
}
func (dp *DatabaseProvider) SetQuery(c int) error {
	_, err := dp.db.Exec("UPDATE counter SET c=$1", c)
	if err != nil {
		return err
	}

	return nil
}

func (dp *DatabaseProvider) ClearQuery() error {
	_, err := dp.db.Exec("DELETE FROM counter")
	if err != nil {
		return err
	}

	return nil
}

// Обработчики HTTP-запросов
func (h *Handlers) GetCounter(w http.ResponseWriter, r *http.Request) {
	counter, _ := h.dbProvider.SelectQuery()
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Счётчик сейчас %d", counter)))
}
func (h *Handlers) PostCounter(w http.ResponseWriter, r *http.Request) {
	counter, _ := h.dbProvider.SelectQuery()
	var err error

	counter += 1
	if counter > 1 {
		h.dbProvider.SetQuery(counter)
	} else {
		err = h.dbProvider.InsertQuery(counter)
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func (h *Handlers) SetCounter(w http.ResponseWriter, r *http.Request) {
	var err error
	var int_num int

	_, err = h.dbProvider.SelectQuery()
	if err != nil {
		_ = h.dbProvider.InsertQuery(0)
	}
	num := r.URL.Query().Get("num")
	if num == "" {
		int_num, err = h.dbProvider.SelectQuery()
	} else {
		int_num, err = strconv.Atoi(num)
	}

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("num должен быть целым числом"))
	} else {
		err = h.dbProvider.SetQuery(int_num)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf("Значение %d установлено", int_num)))
		}
	}
}

func (h *Handlers) ClearCounter(w http.ResponseWriter, r *http.Request) {
	err := h.dbProvider.ClearQuery()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Счетик сброшен..."))
}

func main() {
	// Считываем аргументы командной строки
	address := flag.String("address", "127.0.0.1:8083", "Адрес для запуска сервиса Counter")
	flag.Parse()

	// Формирование строки подключения для postgres
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	// Создание соединения с сервером postgres
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Создаем провайдер для БД с набором методов
	dp := DatabaseProvider{db: db}
	// Создаем экземпляр структуры с набором обработчиков
	h := Handlers{dbProvider: dp}

	// Регистрируем обработчики
	http.HandleFunc("/get", corsMiddleware(h.GetCounter))
	http.HandleFunc("/post", corsMiddleware(h.PostCounter))
	http.HandleFunc("/clear", corsMiddleware(h.ClearCounter))
	http.HandleFunc("/set", corsMiddleware(h.SetCounter))

	// Запускаем веб-сервер на указанном адресе
	err = http.ListenAndServe(*address, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			return
		}

		next(w, r)
	}
}

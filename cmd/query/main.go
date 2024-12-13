package main

// некоторые импорты нужны для проверки
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
func (dp *DatabaseProvider) SelectQuery() (string, int, error) {
	var name string
	var age int

	// Получаем одно сообщение из таблицы query, отсортированной в случайном порядке
	row := dp.db.QueryRow("SELECT name, age FROM query ORDER BY RANDOM() LIMIT 1")
	err := row.Scan(&name, &age)
	if err != nil {
		return "", 0, err
	}

	return name, age, nil
}
func (dp *DatabaseProvider) InsertQuery(name string, age int) error {
	_, err := dp.db.Exec("INSERT INTO query (name, age) VALUES ($1, $2)", name, age)
	if err != nil {
		return err
	}

	return nil
}

func (dp *DatabaseProvider) ClearQuery() error {
	_, err := dp.db.Exec("DELETE FROM query")
	if err != nil {
		return err
	}

	return nil
}

// Обработчики HTTP-запросов
func (h *Handlers) GetQuery(w http.ResponseWriter, r *http.Request) {
	name, age, err := h.dbProvider.SelectQuery()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("Name=%s Age=%d", name, age)))
	}
}
func (h *Handlers) PostQuery(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "Guest"
	}
	age := r.URL.Query().Get("age")
	if age == "" {
		age = "0"
	}
	// Конвертация в тип данных int
	int_age, err := strconv.Atoi(age)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Age должен быть целым числом"))
	}

	err = h.dbProvider.InsertQuery(name, int_age)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *Handlers) ClearQuery(w http.ResponseWriter, r *http.Request) {
	err := h.dbProvider.ClearQuery()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("База данных очищена..."))
}

func main() {
	// Считываем аргументы командной строки
	address := flag.String("address", "127.0.0.1:8082", "Адрес для запуска сервиса Query")
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
	http.HandleFunc("/get", corsMiddleware(h.GetQuery))
	http.HandleFunc("/post", corsMiddleware(h.PostQuery))
	http.HandleFunc("/clear", corsMiddleware(h.ClearQuery))

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

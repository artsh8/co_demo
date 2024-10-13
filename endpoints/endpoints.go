package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/lib/pq"
)

const defaultLimit = 10
const defaultOffset = 0

// Создать соединение с БД
func connectDB() (*sql.DB, error) {
	connStr := "user=customuser password=custompassword dbname=co host=pgpool port=5432 sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return db, nil
}

// Чтнеие коллекции из БД
func queryRowsLimit(query string, limitStr string, offsetStr string) (*sql.Rows, error) {
	db, err := connectDB()
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("ошибка соединения с БД")
	}
	defer db.Close()

	limit := defaultLimit
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	offset := defaultOffset
	if offsetStr != "" {
		parsedOffset, err := strconv.Atoi(offsetStr)
		if err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	rows, err := db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

// Чтение записей по фильтру
func queryRowsFilter(q string, ids ...any) (*sql.Rows, error) {
	db, err := connectDB()
	if err != nil {
		return nil, errors.New("ошибка соединения с БД")
	}
	defer db.Close()

	rows, err := db.Query(q, ids...)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

// Чтение одной записи из БД по id
func queryRow(query string, args ...any) *sql.Row {
	db, err := connectDB()
	if err != nil {
		fmt.Println(err)
	}
	defer db.Close()

	row := db.QueryRow(query, args...)
	return row
}

func getProducts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	query := `
		SELECT p.id, p.name, p.price, p.amount, m.id, m.name, COUNT(*) OVER ()
		FROM product p
		JOIN merchant m ON m.id = p.merchant_id
		ORDER BY p.id ASC
		LIMIT $1 OFFSET $2
		`

	rows, err := queryRowsLimit(query, limitStr, offsetStr)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := ErrorResponse{Error: "Ошибка получения товаров"}
		json.NewEncoder(w).Encode(response)
		fmt.Println(err)
		return
	}
	defer rows.Close()

	var products []Product
	var total int
	for rows.Next() {
		var product Product
		if err := rows.Scan(&product.ID, &product.Name, &product.Price, &product.Amount, &product.Merchant.ID, &product.Merchant.Name, &total); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := ErrorResponse{Error: "Ошибка обработки списка товаров"}
			json.NewEncoder(w).Encode(response)
			return
		}

		products = append(products, product)
	}

	productsV := make([]ProductV, len(products))

	for i, product := range products {
		productsV[i] = ProductV{
			ID:       product.ID,
			Merchant: product.Merchant,
			Name:     product.Name,
			Price:    product.Price.toFloat(),
			Amount:   product.Amount,
		}
	}
	response := struct {
		Total int        `json:"total"`
		Data  []ProductV `json:"data"`
	}{
		Total: total,
		Data:  productsV,
	}

	json.NewEncoder(w).Encode(response)
}

func getProductById(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) > 4 {
		w.WriteHeader(http.StatusBadRequest)
		response := ErrorResponse{Error: "Неправильно указан путь для получения товара по идентификатору"}
		json.NewEncoder(w).Encode(response)
		return
	}

	idStr := parts[3]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := ErrorResponse{Error: "Идентификатор должен быть целым числом"}
		json.NewEncoder(w).Encode(response)
		return
	}
	query := `
		SELECT p.id, p.name, p.price, p.amount, m.id, m.name
		FROM product p
		JOIN merchant m ON m.id = p.merchant_id
		WHERE p.id = $1
		`

	row := queryRow(query, id)
	var product Product
	err = row.Scan(&product.ID, &product.Name, &product.Price, &product.Amount, &product.Merchant.ID, &product.Merchant.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			response := ErrorResponse{Error: "Товар не найден по указанному id"}
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			response := ErrorResponse{Error: "Ошибка при получении товара"}
			json.NewEncoder(w).Encode(response)
		}
		return
	}
	productV := ProductV{
		ID:       product.ID,
		Merchant: product.Merchant,
		Name:     product.Name,
		Price:    product.Price.toFloat(),
		Amount:   product.Amount,
	}

	json.NewEncoder(w).Encode(productV)
}

func getOrders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	query := `
		SELECT co.id, c.id, c.first_name, c.last_name, c.email, COUNT(*) OVER() AS total
		FROM customer_order co
		JOIN customer c ON c.id = co.customer_id
		ORDER BY co.id ASC
		LIMIT $1 OFFSET $2
		`

	rows, err := queryRowsLimit(query, limitStr, offsetStr)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := ErrorResponse{Error: "Ошибка получения заказов"}
		json.NewEncoder(w).Encode(response)
		return
	}
	defer rows.Close()

	var orders []Order
	var total int
	for rows.Next() {
		var order Order
		if err := rows.Scan(&order.ID, &order.Customer.ID, &order.Customer.FirstName, &order.Customer.LastName, &order.Customer.Email, &total); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := ErrorResponse{Error: "Ошибка обработки списка заказов"}
			json.NewEncoder(w).Encode(response)
			return
		}

		orders = append(orders, order)
	}

	query = `
	SELECT p.id, po.id, p.name, po.amount, p.price, m.id, m.name
	FROM product_order po
	JOIN product p ON p.id = po.product_id
	JOIN merchant m ON m.id = p.merchant_id
	WHERE po.order_id = ANY($1)
	`
	var orderIds []int
	for _, order := range orders {
		orderIds = append(orderIds, order.ID)
	}

	rows, err = queryRowsFilter(query, pq.Array(orderIds))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := ErrorResponse{Error: "Ошибка получения покупателей"}
		json.NewEncoder(w).Encode(response)
		return
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var product Product
		if err := rows.Scan(&product.ID, &product.OrderID, &product.Name, &product.Amount, &product.Price, &product.Merchant.ID, &product.Merchant.Name); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := ErrorResponse{Error: "Ошибка обработки списка товаров"}
			json.NewEncoder(w).Encode(response)
			return
		}

		products = append(products, product)
	}

	productsMap := make(map[int][]Product, len(products))
	for _, product := range products {
		productsMap[product.OrderID] = append(productsMap[product.OrderID], product)
	}

	ordersV := make([]OrderV, 0, len(orders))
	for _, o := range orders {
		o.Products = productsMap[o.ID]
		o.calcCheckout()
		oV := o.toOrderV()
		ordersV = append(ordersV, oV)
	}

	response := struct {
		Total int      `json:"total"`
		Data  []OrderV `json:"data"`
	}{
		Total: total,
		Data:  ordersV,
	}
	json.NewEncoder(w).Encode(response)
}

func deleteProduct(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) > 4 {
		w.WriteHeader(http.StatusBadRequest)
		response := ErrorResponse{Error: "Неправильно указан путь для удаления товара по идентификатору"}
		json.NewEncoder(w).Encode(response)
		return
	}

	idStr := parts[3]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := ErrorResponse{Error: "Идентификатор должен быть целым числом"}
		json.NewEncoder(w).Encode(response)
		return
	}

	query := "DELETE FROM product WHERE id = $1 RETURNING id"
	var deletedID int
	err = queryRow(query, id).Scan(&deletedID)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusBadRequest)
			response := ErrorResponse{Error: "Отсутсвует запись для удаления"}
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			response := ErrorResponse{Error: "Ошибка при удалении"}
			json.NewEncoder(w).Encode(response)
		}
		return
	}

	response := MessageResponse{
		Message: fmt.Sprintf("Удалена запись с id: %d", deletedID),
	}

	json.NewEncoder(w).Encode(response)
}

func deleteOrder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) > 4 {
		w.WriteHeader(http.StatusBadRequest)
		response := ErrorResponse{Error: "Неправильно указан путь для удаления заказа по идентификатору"}
		json.NewEncoder(w).Encode(response)
		return
	}

	idStr := parts[3]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := ErrorResponse{Error: "Идентификатор должен быть целым числом"}
		json.NewEncoder(w).Encode(response)
		return
	}

	query := "DELETE FROM customer_order WHERE id = $1 RETURNING id"
	var deletedID int
	err = queryRow(query, id).Scan(&deletedID)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusBadRequest)
			response := ErrorResponse{Error: "Отсутсвует запись для удаления"}
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			response := ErrorResponse{Error: "Ошибка при удалении"}
			json.NewEncoder(w).Encode(response)
		}
		return
	}

	response := MessageResponse{
		Message: fmt.Sprintf("Удалена запись с id: %d", deletedID),
	}

	json.NewEncoder(w).Encode(response)
}

func getRestock(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	query := `
		SELECT p.id, SUM(po.amount) - p.amount AS restock, COUNT(*) OVER () AS total
		FROM product_order po 
		JOIN product p ON p.id = po.product_id
		GROUP BY p.id
		HAVING p.amount - SUM(po.amount) <= 0
		LIMIT $1 OFFSET $2
		`

	rows, err := queryRowsLimit(query, limitStr, offsetStr)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := ErrorResponse{Error: "Ошибка получения информации для пополнения запасов"}
		json.NewEncoder(w).Encode(response)
		return
	}
	defer rows.Close()

	var restocks []ProductRestock
	var total int
	for rows.Next() {
		var restock ProductRestock
		if err := rows.Scan(&restock.ProductID, &restock.Amount, &total); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := ErrorResponse{Error: "Ошибка обработки информации для пополнения запасов"}
			json.NewEncoder(w).Encode(response)
			return
		}

		restocks = append(restocks, restock)
	}

	response := struct {
		Total int              `json:"total"`
		Data  []ProductRestock `json:"data"`
	}{
		Total: total,
		Data:  restocks,
	}
	json.NewEncoder(w).Encode(response)
}

func updateStock(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) > 4 {
		w.WriteHeader(http.StatusBadRequest)
		response := ErrorResponse{Error: "Неправильно указан путь для изменения количества товара"}
		json.NewEncoder(w).Encode(response)
		return
	}

	idStr := parts[3]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := ErrorResponse{Error: "Идентификатор должен быть целым числом"}
		json.NewEncoder(w).Encode(response)
		return
	}

	var amount StockAmount
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()

	if err := decoder.Decode(&amount); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := ErrorResponse{Error: "Ошибка обработки тела запроса"}
		json.NewEncoder(w).Encode(response)
		return
	}
	validate := validator.New()
	if err := validate.Struct(amount); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := ErrorResponse{Error: "Некорректное тело запроса"}
		json.NewEncoder(w).Encode(response)
		return
	}

	query := "UPDATE product SET amount = $1 WHERE id = $2 RETURNING id"
	var updatedProduct int
	err = queryRow(query, amount.Amount, id).Scan(&updatedProduct)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusBadRequest)
			response := ErrorResponse{Error: "Отсутсвует запись товара для изменения его количетсва"}
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			response := ErrorResponse{Error: "Ошибка при обновлении информации о количестве товара"}
			json.NewEncoder(w).Encode(response)
		}
		return
	}

	response := MessageResponse{
		Message: fmt.Sprintf("Изменено количество товара с id: %d", updatedProduct),
	}

	json.NewEncoder(w).Encode(response)
}

func ping(w http.ResponseWriter, r *http.Request) {
	response := MessageResponse{Message: "pong"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	http.HandleFunc("GET /v1/products", getProducts)
	http.HandleFunc("GET /v1/products/", getProductById)
	http.HandleFunc("DELETE /v1/products/", deleteProduct)
	http.HandleFunc("PATCH /v1/products/", updateStock)
	http.HandleFunc("GET /v1/orders", getOrders)
	http.HandleFunc("DELETE /v1/orders/", deleteOrder)
	http.HandleFunc("GET /v1/restock", getRestock)
	http.HandleFunc("GET /ping", ping)

	fmt.Println("Starting server on :8081...")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal(err)
	}
}

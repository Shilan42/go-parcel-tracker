package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

// ParcelStore - структура для работы с посылками в базе данных
type ParcelStore struct {
	db *sql.DB
}

// NewParcelStore - конструктор для создания нового экземпляра ParcelStore (В ней поле для хранения подключения к базе данных)
func NewParcelStore(db *sql.DB) ParcelStore {
	return ParcelStore{db: db}
}

// Add - метод для добавления новой посылки в базу данных
func (s ParcelStore) Add(p Parcel) (int, error) {

	// Выполняем SQL-запрос на вставку новой посылки
	res, err := s.db.Exec("INSERT INTO parcel (client, status, address, created_at) VALUES (:client, :status, :address, :created_at)",
		sql.Named("client", p.Client),
		sql.Named("status", p.Status),
		sql.Named("address", p.Address),
		sql.Named("created_at", p.CreatedAt))
	if err != nil {
		return 0, fmt.Errorf("failed to add parcel to the database: client=%d, status=%s, address=%s, error: %w", p.Client, p.Status, p.Address, err)
	}

	// Получаем ID добавленной посылки
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get ID of the added parcel: error: %w", err)
	}
	// Возвращаем ID новой посылки
	return int(id), nil
}

// Get - метод для получения посылки по её номеру
func (s ParcelStore) Get(number int) (Parcel, error) {

	// Создаем пустую структуру посылки
	p := Parcel{}

	// Выполняем SQL-запрос для получения данных о посылке
	row := s.db.QueryRow("SELECT * FROM parcel WHERE number = :number", sql.Named("number", number))

	// Сканируем результат запроса и записываем его в структуру посылки
	err := row.Scan(&p.Number, &p.Client, &p.Status, &p.Address, &p.CreatedAt)
	if err != nil {
		return p, fmt.Errorf("failed to retrieve parcel with number %d: error: %w", number, err)
	}

	// Возвращаем найденную посылку
	return p, nil
}

// GetByClient - метод для получения всех посылок определенного клиента
func (s ParcelStore) GetByClient(client int) ([]Parcel, error) {

	// Создаем слайс для хранения найденных посылок
	var res []Parcel

	// Выполняем SQL-запрос для получения всех посылок клиента
	rows, err := s.db.Query("SELECT * FROM parcel WHERE client = :client", sql.Named("client", client))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve client's parcels %d: error: %w", client, err)
	}
	// Закрываем результат запроса после использования
	defer rows.Close()

	// Итерируемся по всем строкам результата
	for rows.Next() {
		// Создаем новую структуру посылки
		p := Parcel{}

		// Сканируем данные текущей строки и записываем их в структуру посылки
		err = rows.Scan(&p.Number, &p.Client, &p.Status, &p.Address, &p.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("row scanning error while retrieving client's parcels %d: error: %w", client, err)
		}
		// Добавляем посылку = структуру в слайс для хранения найденных посылок
		res = append(res, p)
	}

	// Проверяем наличие ошибок, возникших при итерации по всем строкам результата запроса
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating through rows while retrieving client's parcels %d: %w", client, err)
	}

	// Возвращаем все найденные посылки
	return res, nil
}

// SetStatus - метод для обновления статуса посылки
func (s ParcelStore) SetStatus(number int, status string) error {

	// Выполняем SQL-запрос на обновление статуса
	_, err := s.db.Exec("UPDATE parcel SET status = :status WHERE number = :number", sql.Named("status", status), sql.Named("number", number))
	if err != nil {
		return fmt.Errorf("failed to update parcel status №%d to '%s': error: %w", number, status, err)
	}
	// Возвращаем nil при успешном выполнении
	return nil
}

// SetAddress - метод для установки нового адреса посылки при условии, что её статус зарегистрирован
func (s ParcelStore) SetAddress(number int, address string) error {

	// Выполняем обновление с проверкой статуса в одном запросе
	result, err := s.db.Exec("UPDATE parcel SET address = :address WHERE number = :number AND status = :status",
		sql.Named("address", address),
		sql.Named("number", number),
		sql.Named("status", ParcelStatusRegistered))
	if err != nil {
		return fmt.Errorf("address update error for parcel №%d: new address '%s', error: %w", number, address, err)
	}

	// Проверяем, что строка была обновлена
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		log.Printf("update denied for parcel %d: invalid status or parcel not found", number)
		return nil
	}

	return nil
}

// SetAddress - метод для удаления посылки из базы данных при условии, что её статус зарегистрирован
func (s ParcelStore) Delete(number int) error {

	// Выполняем удаление с проверкой статуса в одном запросе
	result, err := s.db.Exec(
		"DELETE FROM parcel WHERE number = :number AND status = :status",
		sql.Named("number", number),
		sql.Named("status", ParcelStatusRegistered),
	)
	if err != nil {
		return fmt.Errorf("parcel deletion error №%d: %w", number, err)
	}

	// Проверяем, что строка была удалена
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		log.Printf("delete denied for parcel №%d: invalid status or parcel not found", number)
		return nil
	}

	return nil
}

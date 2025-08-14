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

// SetAddress - метод для установки нового адреса посылки
func (s ParcelStore) SetAddress(number int, address string) error {
	// Начинаем транзакцию для обеспечения целостности данных (решил попробовать поюзать транзакции)
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("transaction start error for updating parcel №%d address: %w", number, err)
	}
	// Автооткат транзакции при ошибке
	defer tx.Rollback()

	// Получаем текущую посылку для проверки статуса
	p, err := s.Get(number)
	if err != nil {
		return fmt.Errorf("error retrieving parcel №%d data for status check: %w", number, err)
	}

	// Проверяем, что статус посылки позволяет изменить адрес
	if p.Status != ParcelStatusRegistered {
		log.Printf("impossible to update address for parcel №%d: invalid status (expected 'registered', received: %s)", number, p.Status)
		// Возвращаем nil, так как это не критичная ошибка
		return nil
	}

	// Выполняем обновление адреса
	_, err = s.db.Exec("UPDATE parcel SET address = :address WHERE number = :number", sql.Named("address", address), sql.Named("number", number))
	if err != nil {
		return fmt.Errorf("address update error for parcel №%d: new address '%s', error: %w", number, address, err)
	}

	// Завершаем транзакцию
	return tx.Commit()
}

func (s ParcelStore) Delete(number int) error {
	// Начинаем транзакцию для обеспечения целостности данных (решил попробовать поюзать транзакции)
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("transaction start error for parcel deletion №%d: %w", number, err)
	}
	// Автооткат при ошибке
	defer tx.Rollback()

	// Получаем текущую посылку для проверки статуса
	p, err := s.Get(number)
	if err != nil {
		return fmt.Errorf("error retrieving parcel data №%d for status check: %w", number, err)
	}

	// Проверяем, что статус посылки позволяет её удалить
	if p.Status != ParcelStatusRegistered {
		log.Printf("impossible to delete parcel №%d: invalid status (expected 'registered', received: %s)", number, p.Status)
		// Возвращаем nil, так как это не критичная ошибка
		return nil
	}

	// Выполняем удаление посылки
	_, err = s.db.Exec("DELETE FROM parcel WHERE number = :number", sql.Named("number", number))
	if err != nil {
		return fmt.Errorf("parcel deletion error №%d: %w", number, err)
	}

	// Завершаем транзакцию
	return tx.Commit()
}

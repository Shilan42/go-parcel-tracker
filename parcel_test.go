package main

import (
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

var (
	// randSource источник псевдо случайных чисел.
	// Для повышения уникальности в качестве seed
	// используется текущее время в unix формате (в виде числа)
	randSource = rand.NewSource(time.Now().UnixNano())
	// randRange использует randSource для генерации случайных чисел
	randRange = rand.New(randSource)
)

// getTestParcel возвращает тестовую посылку
func getTestParcel() Parcel {
	return Parcel{
		Client:    1000,
		Status:    ParcelStatusRegistered,
		Address:   "test",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

// cleanDatabase - очистка базы данных от записей
func cleanDatabase(db *sql.DB) error {
	// Выполнение SQL запроса на удаление всех записей
	_, err := db.Exec("DELETE FROM parcel")
	if err != nil {
		return fmt.Errorf("failed to execute DELETE operation on 'parcel' table. Error details: %w", err)
	}
	return nil
}

// setupDatabase - настройка подключения к базе данных
func setupDatabase(t *testing.T) *sql.DB {
	// Подключение к SQLite базе данных
	db, err := sql.Open("sqlite", "tracker_test.db")
	require.NoError(t, err, "failed to establish database connection: tracker_test.db. Error details: %w", err)

	// Очистка БД перед каждым тестом
	err = cleanDatabase(db)
	require.NoError(t, err, err)

	// Возврат подключенной базы данных
	return db
}

// TestAddGetDelete - тест для проверки операций создания, получения и удаления посылки
func TestAddGetDelete(t *testing.T) {
	// Подготовка окружения и автоматическое закрытие БД после теста
	db := setupDatabase(t)
	defer db.Close()

	// Создание хранилища посылок и получение тестовой посылки
	store := NewParcelStore(db)
	parcel := getTestParcel()
	var err error

	// Структура для хранения тестовых кейсов
	tests := []struct {
		name     string                                // Название тестового кейса
		testFunc func(*testing.T, ParcelStore, Parcel) // Функция, реализующая логику теста
	}{
		{
			name: "Parcel insertion test",
			testFunc: func(*testing.T, ParcelStore, Parcel) {
				parcel.Number, err = store.Add(parcel)
				assert.NotEmpty(t, parcel.Number, "parcel ID should not be empty after insertion. Test parcel: %v", parcel)
				require.NoError(t, err, "failed to insert parcel into database. Parcel details: %v. Error: %v", parcel, err)
			},
		},
		{
			name: "Parcel retrieval test by ID",
			testFunc: func(*testing.T, ParcelStore, Parcel) {
				res, err := store.Get(parcel.Number)
				require.NoError(t, err, "failed to retrieve parcel with ID %d from database. Error: %v", parcel.Number, err)

				assert.Equal(t, res.Address, parcel.Address, "address mismatch. Expected: %v, Actual: %v", parcel.Address, res.Address)
				assert.Equal(t, res.Client, parcel.Client, "client mismatch. Expected: %v, Actual: %v", parcel.Client, res.Client)
				assert.Equal(t, res.CreatedAt, parcel.CreatedAt, "createdAt mismatch. Expected: %v, Actual: %v", parcel.CreatedAt, res.CreatedAt)
				assert.Equal(t, res.Number, parcel.Number, "ID mismatch. Expected: %v, Actual: %v", parcel.Number, res.Number)
				assert.Equal(t, res.Status, parcel.Status, "status mismatch. Expected: %v, Actual: %v", parcel.Status, res.Status)
			},
		},
		{
			name: "Parcel deletion test",
			testFunc: func(*testing.T, ParcelStore, Parcel) {
				err = store.Delete(parcel.Number)
				require.NoError(t, err, "failed to delete parcel with ID %d from database", parcel.Number)

				_, err = store.Get(parcel.Number)
				require.Error(t, err, "expected error when trying to retrieve deleted parcel with ID %d", parcel.Number)

				originalErr := errors.Unwrap(err)
				require.Equal(t, sql.ErrNoRows, originalErr, "expected specific sql.ErrNoRows error when searching for deleted parcel with ID %d", parcel.Number)
			},
		},
	}
	// Итерируемся по всем тестовым кейсам
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc(t, store, parcel)
		})
	}
}

// TestSetAddress - тест для проверки операции обновления адреса посылки
func TestSetAddress(t *testing.T) {
	// Подготовка окружения и автоматическое закрытие БД после теста
	db := setupDatabase(t)
	defer db.Close()

	// Создание хранилища посылок и получение тестовой посылки
	store := NewParcelStore(db)
	parcel := getTestParcel()
	// Новый адрес для обновления
	newAddress := "new test address"
	var err error

	// Структура для хранения тестовых кейсов
	tests := []struct {
		name     string                                // Название тестового кейса
		testFunc func(*testing.T, ParcelStore, Parcel) // Функция, реализующая логику теста
	}{
		{
			name: "Parcel insertion test",
			testFunc: func(*testing.T, ParcelStore, Parcel) {
				parcel.Number, err = store.Add(parcel)
				require.NoError(t, err, "failed to insert parcel into database. Parcel details: %v. Error: %v", parcel, err)
				assert.NotEmpty(t, parcel.Number, "parcel ID should not be empty after insertion. Test parcel: %v", parcel)
			},
		},
		{
			name: "Parcel address update test",
			testFunc: func(*testing.T, ParcelStore, Parcel) {
				err := store.SetAddress(parcel.Number, newAddress)
				require.NoError(t, err, "failed to update address for parcel with ID %d. New address: %s. Error: %w", parcel.Number, newAddress, err)
			},
		},
		{
			name: "Parcel verify address update correctness",
			testFunc: func(*testing.T, ParcelStore, Parcel) {
				res, err := store.Get(parcel.Number)
				require.NoError(t, err, "failed to retrieve parcel with ID %d from database. Error: %v", parcel.Number, err)
				assert.Equal(t, res.Address, newAddress, "address update verification failed. Expected address: %s, Actual address: %s", newAddress, res.Address)
			},
		},
	}
	// Итерируемся по всем тестовым кейсам
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc(t, store, parcel)
		})
	}
}

// TestSetStatus - тест для проверки операции обновления статуса посылки
func TestSetStatus(t *testing.T) {
	// Подготовка окружения и автоматическое закрытие БД после теста
	db := setupDatabase(t)
	defer db.Close()

	// Создание хранилища посылок и получение тестовой посылки
	store := NewParcelStore(db)
	parcel := getTestParcel()
	var err error

	// Структура для хранения тестовых кейсов
	tests := []struct {
		name     string                                // Название тестового кейса
		testFunc func(*testing.T, ParcelStore, Parcel) // Функция, реализующая логику теста
	}{
		{
			name: "Parcel insertion test",
			testFunc: func(*testing.T, ParcelStore, Parcel) {
				parcel.Number, err = store.Add(parcel)
				require.NoError(t, err, "failed to insert parcel into database. Parcel details: %v. Error: %v", parcel, err)
				assert.NotEmpty(t, parcel.Number, "parcel ID should not be empty after insertion. Test parcel: %v", parcel)
			},
		},
		{
			name: "Parcel status update test",
			testFunc: func(*testing.T, ParcelStore, Parcel) {
				err := store.SetStatus(parcel.Number, ParcelStatusSent)
				require.NoError(t, err, "failed to update status for parcel with ID %d. Status: %s. Error: %w", parcel.Number, ParcelStatusSent, err)
			},
		},
		{
			name: "Parcel verify status update correctness",
			testFunc: func(*testing.T, ParcelStore, Parcel) {
				res, err := store.Get(parcel.Number)
				require.NoError(t, err, "failed to retrieve parcel with ID %d from database. Error: %v", parcel.Number, err)
				assert.Equal(t, res.Status, ParcelStatusSent, "status update verification failed. Expected status: %s, Actual status: %s", ParcelStatusSent, res.Status)
			},
		},
	}
	// Итерируемся по всем тестовым кейсам
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc(t, store, parcel)
		})
	}
}

// TestGetByClient - тест для проверки получения списка посылок по идентификатору клиента
func TestGetByClient(t *testing.T) {
	// Подготовка окружения и автоматическое закрытие БД после теста
	db := setupDatabase(t)
	defer db.Close()

	// Создание хранилища посылок и слайса тестовых посылок
	store := NewParcelStore(db)
	parcels := []Parcel{
		getTestParcel(),
		getTestParcel(),
		getTestParcel(),
	}
	// Мапа для хранения добавленных посылок в БД. Используется при сравнении добавленных данных с исходными
	parcelMap := map[int]Parcel{}

	// Генерируем случайное ID клиента и задаём всем посылкам один и тот же идентификатор клиента
	client := randRange.Intn(10_000_000)
	parcels[0].Client = client
	parcels[1].Client = client
	parcels[2].Client = client

	// Добавление посылок в базу данных
	for i := 0; i < len(parcels); i++ {
		id, err := store.Add(parcels[i])
		require.NoError(t, err, "failed to insert parcel into database. Parcel details: %v. Error: %v", parcels[i], err)
		assert.NotEmpty(t, id, "parcel ID should not be empty after insertion. Test parcel: %v", parcels[i])

		parcels[i].Number = id     // Обновление ID посылки
		parcelMap[id] = parcels[i] // Сохранение посылки в мапу
	}

	// Получение посылок по ID клиента
	storedParcels, err := store.GetByClient(client)
	require.NoError(t, err, "failed to retrieve parcels for client with ID: %d. Error: %w", client, err)
	assert.Equal(t, len(storedParcels), len(parcelMap), "mismatch in retrieved parcel count. Expected: %d, Actual: %d", len(parcelMap), len(storedParcels))

	// Проверка корректности полученных данных
	for _, parcel := range storedParcels {
		originalParcel, ok := parcelMap[parcel.Number]
		require.True(t, ok, "parcel with ID %d not found in original data", parcel.Number)
		// Проверка всех полей полученной посылки
		assert.Equal(t, parcel.Address, originalParcel.Address, "address mismatch. Expected: %s, Actual: %s", originalParcel.Address, parcel.Address)
		assert.Equal(t, parcel.Client, originalParcel.Client, "client ID mismatch. Expected: %d, Actual: %d", originalParcel.Client, parcel.Client)
		assert.Equal(t, parcel.CreatedAt, originalParcel.CreatedAt, "createdAt mismatch. Expected: %s, Actual: %s", originalParcel.CreatedAt, parcel.CreatedAt)
		assert.Equal(t, parcel.Number, originalParcel.Number, "parcel ID mismatch. Expected: %d, Actual: %d", originalParcel.Number, parcel.Number)
		assert.Equal(t, parcel.Status, originalParcel.Status, "status mismatch. Expected: %s, Actual: %s", originalParcel.Status, parcel.Status)
	}
}

package main

import (
	"database/sql"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/go-sql-driver/mysql"
)

var (
	randSource = rand.NewSource(time.Now().UnixNano())
	randRange  = rand.New(randSource)
)

func getTestParcel() Parcel {
	return Parcel{
		Client:    1000,
		Status:    ParcelStatusRegistered,
		Address:   "test",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

func setupDB(t *testing.T) *sql.DB {
	db, err := sql.Open("mysql", "user:password@/dbname")
	require.NoError(t, err)

	err = db.Ping()
	require.NoError(t, err)

	// Очищаем таблицу перед тестами
	_, err = db.Exec("DELETE FROM parcel")
	require.NoError(t, err)

	return db
}

func TestAddGetDelete(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	store := NewParcelStore(db)
	parcel := getTestParcel()

	// Add
	id, err := store.Add(parcel)
	assert.NoError(t, err)
	assert.NotZero(t, id)

	// Get
	storedParcel, err := store.Get(id)
	assert.NoError(t, err)
	assert.Equal(t, id, storedParcel.Number)  // Проверяем Number
	assert.Equal(t, parcel.Client, storedParcel.Client)
	assert.Equal(t, parcel.Status, storedParcel.Status)
	assert.Equal(t, parcel.Address, storedParcel.Address)
	assert.Equal(t, parcel.CreatedAt, storedParcel.CreatedAt)

	// Delete
	err = store.Delete(id)
	assert.NoError(t, err)

	// Verify delete
	_, err = store.Get(id)
	assert.Error(t, err)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestSetAddress(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	store := NewParcelStore(db)
	parcel := getTestParcel()

	// Add
	id, err := store.Add(parcel)
	assert.NoError(t, err)

	// Set address
	newAddress := "new test address"
	err = store.SetAddress(id, newAddress)
	assert.NoError(t, err)

	// Check
	updatedParcel, err := store.Get(id)
	assert.NoError(t, err)
	assert.Equal(t, newAddress, updatedParcel.Address)
}

func TestSetStatus(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	store := NewParcelStore(db)
	parcel := getTestParcel()

	// Add
	id, err := store.Add(parcel)
	assert.NoError(t, err)

	// Set status
	newStatus := ParcelStatusSent
	err = store.SetStatus(id, newStatus)
	assert.NoError(t, err)

	// Check
	updatedParcel, err := store.Get(id)
	assert.NoError(t, err)
	assert.Equal(t, newStatus, updatedParcel.Status)
}

func TestGetByClient(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	store := NewParcelStore(db)
	parcels := []Parcel{
		getTestParcel(),
		getTestParcel(),
		getTestParcel(),
	}
	parcelMap := map[int]Parcel{}

	client := randRange.Intn(10_000_000)
	parcels[0].Client = client
	parcels[1].Client = client
	parcels[2].Client = client

	// Add
	for i := 0; i < len(parcels); i++ {
		id, err := store.Add(parcels[i])
		assert.NoError(t, err)
		assert.NotZero(t, id)

		parcels[i].Number = id
		parcelMap[id] = parcels[i]
	}

	// Get by client
	storedParcels, err := store.GetByClient(client)
	assert.NoError(t, err)
	assert.Len(t, storedParcels, len(parcels))

	// Check
	for _, parcel := range storedParcels {
		expectedParcel, ok := parcelMap[parcel.Number]
		assert.True(t, ok)
		assert.Equal(t, expectedParcel, parcel)  // Проверяем всю структуру
	}
}

func TestSetAddressInvalidStatus(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	store := NewParcelStore(db)
	parcel := getTestParcel()

	// Add and set status to sent
	id, err := store.Add(parcel)
	assert.NoError(t, err)
	err = store.SetStatus(id, ParcelStatusSent)
	assert.NoError(t, err)

	// Try to set address
	err = store.SetAddress(id, "new address")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "нельзя изменить адрес")
}

func TestDeleteInvalidStatus(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	store := NewParcelStore(db)
	parcel := getTestParcel()

	// Add and set status to sent
	id, err := store.Add(parcel)
	assert.NoError(t, err)
	err = store.SetStatus(id, ParcelStatusSent)
	assert.NoError(t, err)

	// Try to delete
	err = store.Delete(id)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "нельзя удалить посылку")
}
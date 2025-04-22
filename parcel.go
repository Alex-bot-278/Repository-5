package main

import (
	"database/sql"
	"errors"
	"fmt"
)

type ParcelStore struct {
	db *sql.DB
}

func NewParcelStore(db *sql.DB) ParcelStore {
	return ParcelStore{db: db}
}

func (s ParcelStore) Add(p Parcel) (int, error) {
	res, err := s.db.Exec(
		"INSERT INTO parcel (client, status, address, created_at) VALUES (?, ?, ?, ?)",
		p.Client, p.Status, p.Address, p.CreatedAt,
	)
	if err != nil {
		return 0, fmt.Errorf("ошибка при добавлении посылки: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("ошибка при получении ID: %w", err)
	}

	return int(id), nil
}

func (s ParcelStore) Get(number int) (Parcel, error) {
	row := s.db.QueryRow(
		"SELECT number, client, status, address, created_at FROM parcel WHERE number = ?",
		number,
	)

	p := Parcel{}
	err := row.Scan(&p.Number, &p.Client, &p.Status, &p.Address, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Parcel{}, fmt.Errorf("посылка с номером %d не найдена", number)
		}
		return Parcel{}, fmt.Errorf("ошибка при сканировании посылки: %w", err)
	}

	return p, nil
}

func (s ParcelStore) GetByClient(client int) ([]Parcel, error) {
	rows, err := s.db.Query(
		"SELECT number, client, status, address, created_at FROM parcel WHERE client = ?",
		client,
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка при запросе посылок: %w", err)
	}
	defer rows.Close()

	var res []Parcel
	for rows.Next() {
		var p Parcel
		if err := rows.Scan(&p.Number, &p.Client, &p.Status, &p.Address, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("ошибка при сканировании строки: %w", err)
		}
		res = append(res, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при обработке результатов: %w", err)
	}

	return res, nil
}

func (s ParcelStore) SetStatus(number int, status string) error {
	res, err := s.db.Exec(
		"UPDATE parcel SET status = ? WHERE number = ?",
		status, number,
	)
	if err != nil {
		return fmt.Errorf("ошибка при обновлении статуса: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка при проверке обновления: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("посылка с номером %d не найдена", number)
	}

	return nil
}

func (s ParcelStore) SetAddress(number int, address string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("ошибка при начале транзакции: %w", err)
	}
	defer tx.Rollback()

	var currentStatus string
	err = tx.QueryRow(
		"SELECT status FROM parcel WHERE number = ? FOR UPDATE",
		number,
	).Scan(&currentStatus)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("посылка с номером %d не найдена", number)
		}
		return fmt.Errorf("ошибка при проверке статуса: %w", err)
	}

	if currentStatus != ParcelStatusRegistered {
		return fmt.Errorf("нельзя изменить адрес: статус должен быть %q, текущий: %q", 
			ParcelStatusRegistered, currentStatus)
	}

	_, err = tx.Exec(
		"UPDATE parcel SET address = ? WHERE number = ?",
		address, number,
	)
	if err != nil {
		return fmt.Errorf("ошибка при обновлении адреса: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ошибка при коммите транзакции: %w", err)
	}

	return nil
}

func (s ParcelStore) Delete(number int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("ошибка при начале транзакции: %w", err)
	}
	defer tx.Rollback()

	var currentStatus string
	err = tx.QueryRow(
		"SELECT status FROM parcel WHERE number = ? FOR UPDATE",
		number,
	).Scan(&currentStatus)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("посылка с номером %d не найдена", number)
		}
		return fmt.Errorf("ошибка при проверке статуса: %w", err)
	}

	if currentStatus != ParcelStatusRegistered {
		return fmt.Errorf("нельзя удалить посылку: статус должен быть %q, текущий: %q",
			ParcelStatusRegistered, currentStatus)
	}

	_, err = tx.Exec(
		"DELETE FROM parcel WHERE number = ?",
		number,
	)
	if err != nil {
		return fmt.Errorf("ошибка при удалении посылки: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ошибка при коммите транзакции: %w", err)
	}

	return nil
}

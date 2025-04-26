package main

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	db *sql.DB
}

func NewDatabase(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть базу данных: %v", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("не удалось подключиться к базе данных: %v", err)
	}

	database := &Database{db: db}
	if err := database.createTables(); err != nil {
		return nil, fmt.Errorf("не удалось создать таблицы: %v", err)
	}

	return database, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY,
			first_name TEXT,
			last_name TEXT,
			username TEXT,
			contact TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS devices (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT,
			price REAL,
			seller_id INTEGER,
			seller_name TEXT,
			contact TEXT,
			category TEXT,
			FOREIGN KEY (seller_id) REFERENCES users(id)
		)`,
	}

	for _, query := range queries {
		_, err := d.db.Exec(query)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *Database) SaveUser(user User) error {
	query := `INSERT OR REPLACE INTO users (id, first_name, last_name, username, contact) 
              VALUES (?, ?, ?, ?, ?)`
	
	_, err := d.db.Exec(query, user.ID, user.FirstName, user.LastName, user.Username, user.Contact)
	return err
}

func (d *Database) GetUsers() (map[int64]User, error) {
	query := `SELECT id, first_name, last_name, username, contact FROM users`
	
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make(map[int64]User)
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.FirstName, &user.LastName, &user.Username, &user.Contact); err != nil {
			return nil, err
		}
		users[user.ID] = user
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (d *Database) SaveDevice(device Device) (int, error) {
	query := `INSERT INTO devices (name, description, price, seller_id, seller_name, contact, category) 
              VALUES (?, ?, ?, ?, ?, ?, ?)`
	
	result, err := d.db.Exec(query, device.Name, device.Description, device.Price, 
		device.SellerID, device.SellerName, device.Contact, device.Category)
	if err != nil {
		return 0, err
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	
	return int(id), nil
}

func (d *Database) GetDevices() ([]Device, error) {
	query := `SELECT id, name, description, price, seller_id, seller_name, contact, category FROM devices`
	
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []Device
	for rows.Next() {
		var device Device
		if err := rows.Scan(&device.ID, &device.Name, &device.Description, &device.Price, 
			&device.SellerID, &device.SellerName, &device.Contact, &device.Category); err != nil {
			return nil, err
		}
		devices = append(devices, device)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return devices, nil
}

func (d *Database) GetDevicesByCategory(category string) ([]Device, error) {
	query := `SELECT id, name, description, price, seller_id, seller_name, contact, category 
              FROM devices WHERE category = ?`
	
	rows, err := d.db.Query(query, category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []Device
	for rows.Next() {
		var device Device
		if err := rows.Scan(&device.ID, &device.Name, &device.Description, &device.Price, 
			&device.SellerID, &device.SellerName, &device.Contact, &device.Category); err != nil {
			return nil, err
		}
		devices = append(devices, device)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return devices, nil
}

func (d *Database) GetDevicesByUser(userID int64) ([]Device, error) {
	query := `SELECT id, name, description, price, seller_id, seller_name, contact, category 
              FROM devices WHERE seller_id = ?`
	
	rows, err := d.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []Device
	for rows.Next() {
		var device Device
		if err := rows.Scan(&device.ID, &device.Name, &device.Description, &device.Price, 
			&device.SellerID, &device.SellerName, &device.Contact, &device.Category); err != nil {
			return nil, err
		}
		devices = append(devices, device)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return devices, nil
}

func (d *Database) GetDeviceByID(deviceID int) (Device, bool, error) {
	query := `SELECT id, name, description, price, seller_id, seller_name, contact, category 
              FROM devices WHERE id = ?`
	
	var device Device
	err := d.db.QueryRow(query, deviceID).Scan(&device.ID, &device.Name, &device.Description, 
		&device.Price, &device.SellerID, &device.SellerName, &device.Contact, &device.Category)
	
	if err == sql.ErrNoRows {
		return Device{}, false, nil
	} else if err != nil {
		return Device{}, false, err
	}
	
	return device, true, nil
}

func (d *Database) RemoveDevice(deviceID int) error {
	query := `DELETE FROM devices WHERE id = ?`
	
	_, err := d.db.Exec(query, deviceID)
	return err
}

func (d *Database) SearchDevices(query string) ([]Device, error) {
	searchQuery := `SELECT id, name, description, price, seller_id, seller_name, contact, category 
                   FROM devices WHERE name LIKE ? OR description LIKE ?`
	
	searchPattern := "%" + query + "%"
	rows, err := d.db.Query(searchQuery, searchPattern, searchPattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []Device
	for rows.Next() {
		var device Device
		if err := rows.Scan(&device.ID, &device.Name, &device.Description, &device.Price, 
			&device.SellerID, &device.SellerName, &device.Contact, &device.Category); err != nil {
			return nil, err
		}
		devices = append(devices, device)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return devices, nil
}

func (d *Database) GetNextDeviceID() (int, error) {
	query := `SELECT MAX(id) FROM devices`
	
	var maxID sql.NullInt64
	err := d.db.QueryRow(query).Scan(&maxID)
	if err != nil {
		return 0, err
	}
	
	if maxID.Valid {
		return int(maxID.Int64) + 1, nil
	}
	
	return 1, nil
} 
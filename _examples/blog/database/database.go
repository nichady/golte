package database

import (
	"database/sql"
	"errors"
	"math/rand"
)

type DB struct {
	db *sql.DB
}

func NewDB(dataSourceName string) *DB {
	db, err := sql.Open("sqlite", dataSourceName)
	if err != nil {
		panic(err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS accounts (
			username STRING PRIMARY KEY,
			password STRING
		);

		CREATE TABLE IF NOT EXISTS sessions (
			id STRING PRIMARY KEY,
			username STRING
		);

		CREATE TABLE IF NOT EXISTS blogs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			author STRING,
			title STRING,
			body STRING,
			timestamp INTEGER DEFAULT (strftime('%s', 'now'))
		);
	`)
	if err != nil {
		panic(err)
	}

	return &DB{db}
}

// RegisterAccount creates a new account from the credentials.
// If the username already exists, [ErrAccountAlreadyExists] is returned.
func (db *DB) RegisterAccount(username, password string) error {
	tx, err := db.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var exists bool
	err = tx.QueryRow("SELECT EXISTS (SELECT 1 FROM accounts WHERE username = $1)", username).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		return ErrAccountAlreadyExists
	}

	_, err = tx.Exec("INSERT INTO accounts (username, password) VALUES ($1, $2)", username, password)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// AccountExists checks if an account exists with the given username and password
func (db *DB) AccountExists(username string, password string) (bool, error) {
	var exists bool
	err := db.db.QueryRow("SELECT EXISTS (SELECT 1 FROM accounts WHERE username = $1 AND password = $2)", username, password).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// CreateSession creates a new session for the specified user and returns the session ID
func (db *DB) CreateSession(username string) (string, error) {
	id := randomString()

	_, err := db.db.Exec("INSERT INTO sessions (id, username) VALUES ($1, $2)", id, username)
	if err != nil {
		return "", err
	}

	return id, nil
}

// GetSession returns the username from the session id.
// If the session does not exist, returns [ErrSessionNotExist].
func (db *DB) GetSession(id string) (string, error) {
	var username string

	err := db.db.QueryRow("SELECT username FROM sessions WHERE id = $1", id).Scan(&username)
	if err == sql.ErrNoRows {
		return "", ErrSessionNotExist
	} else if err != nil {
		return "", err
	}

	return username, nil
}

// PostBlog adds the blog to the database.
func (db *DB) PostBlog(author, title, body string) error {
	_, err := db.db.Exec("INSERT INTO blogs (author, title, body) VALUES ($1, $2, $3)", author, title, body)
	return err
}

// GetBlog gets the blog from the id.
// If it does not exist, returns [ErrBlogNotExist].
func (db *DB) GetBlog(id int) (Blog, error) {
	var blog Blog
	err := db.db.QueryRow("SELECT * FROM blogs WHERE id = $1", id).Scan(&blog.ID, &blog.Author, &blog.Title, &blog.Body, &blog.Timestamp)
	if errors.Is(err, sql.ErrNoRows) {
		return Blog{}, ErrBlogNotExist
	} else if err != nil {
		return Blog{}, err
	}

	return blog, nil
}

// GetAllBlogs returns every blog.
func (db *DB) GetAllBlogs() ([]Blog, error) {
	rows, err := db.db.Query("SELECT * from blogs")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blogs []Blog

	for rows.Next() {
		var blog Blog
		err = rows.Scan(&blog.ID, &blog.Author, &blog.Title, &blog.Body, &blog.Timestamp)
		if err != nil {
			return nil, err
		}

		blogs = append(blogs, blog)
	}

	return blogs, nil
}

// GetUserBlogs returns blogs made from by the specified author.
// Returns [ErrUserNotExist] if the user does not exist.
func (db *DB) GetUserBlogs(username string) ([]Blog, error) {
	tx, err := db.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var exists bool
	err = tx.QueryRow("SELECT EXISTS (SELECT 1 FROM accounts WHERE username = $1)", username).Scan(&exists)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, ErrUserNotExist
	}

	rows, err := tx.Query("SELECT * from blogs WHERE author = $1", username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blogs []Blog
	for rows.Next() {
		var blog Blog
		err = rows.Scan(&blog.ID, &blog.Author, &blog.Title, &blog.Body, &blog.Timestamp)
		if err != nil {
			return nil, err
		}

		blogs = append(blogs, blog)
	}

	return blogs, nil
}

func randomString() string {
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, 20)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

type Blog struct {
	ID        int
	Author    string
	Title     string
	Body      string
	Timestamp int64 // time since unix epoch in seconds
}

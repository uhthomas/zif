package data

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

type Database struct {
	path string
	conn *sql.DB
}

func NewDatabase(path string) *Database {
	var db Database
	db.path = path

	return &db
}

// Connect to a database. If it does not already exist it is created, and the
// proper schema is also setup.
func (db *Database) Connect() error {
	var err error

	db.conn, err = sql.Open("sqlite3", db.path)
	if err != nil {
		return err
	}

	// Enable Write-Ahead Logging
	db.conn.Exec("PRAGMA journal_mode=WAL")

	//db.conn.SetMaxOpenConns(1)

	_, err = db.conn.Exec(sql_create_post_table)
	if err != nil {
		return err
	}

	_, err = db.conn.Exec(sql_create_fts_post)
	if err != nil {
		return err
	}

	_, err = db.conn.Exec(sql_create_upload_date_index)
	if err != nil {
		return err
	}

	return nil
}

// Inserts a piece into the database. All the posts are iterated over and inserted
// within a single SQL transaction.
func (db *Database) InsertPiece(piece *Piece) (err error) {
	tx, err := db.conn.Begin()

	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}

		err = tx.Commit()
	}()

	for _, i := range piece.Posts {
		_, err = tx.Exec(sql_insert_post, i.InfoHash, i.Title, i.Size, i.FileCount,
			i.Seeders, i.Leechers, i.UploadDate, i.Tags)

		if err != nil {
			return
		}
	}

	return
}

// Insert pieces from a channel, good for streaming them from a network or something.
// The fts bool is whether or not a fts index will be generated on every transaction
// commit. Transactions contain 100 pieces, or 100,000 posts.
func (db *Database) InsertPieces(pieces chan *Piece, fts bool) (err error) {
	tx, err := db.conn.Begin()

	if err != nil {
		log.Error(err.Error())
		return err
	}

	n := 0

	defer func() {
		err = tx.Commit()

		if err != nil {
			tx.Rollback()
			log.Error(err.Error())
		}

		close(pieces)
	}()

	//lastId := 0
	for piece := range pieces {
		if piece == nil {
			return nil
		}

		// Insert the transaction every 100,000 posts.
		if n == 99 {
			err = tx.Commit()

			if err != nil {
				log.Error(err.Error())
				return err
			}

			//db.GenerateFts(lastId)
			//lastId = piece.Posts[len(piece.Posts)-1].Id

			tx, err = db.conn.Begin()

			if err != nil {
				log.Error(err.Error())
				return
			}

			n = 0
		}

		for _, i := range piece.Posts {
			_, err = tx.Exec(sql_insert_post, i.InfoHash, i.Title, i.Size, i.FileCount,
				i.Seeders, i.Leechers, i.UploadDate, i.Tags, i.Meta)

			if err != nil {
				log.Error(err.Error())
				return
			}
		}

		n += 1
	}

	return
}

// Insert a single post into the database.
func (db *Database) InsertPost(post Post) (int64, error) {
	// TODO: Is preparing all statements before hand worth doing for perf?
	stmt, err := db.conn.Prepare(sql_insert_post)
	if err != nil {
		return -1, err
	}

	res, err := stmt.Exec(post.InfoHash, post.Title, post.Size, post.FileCount, post.Seeders,
		post.Leechers, post.UploadDate, post.Tags, post.Meta)

	if err != nil {
		return -1, err
	}

	id, err := res.LastInsertId()

	return id, nil
}

// Generate a full text search index since the given id. This should ideally be
// done only for new additions, otherwise on a large dataset it can take a bit of
// time.
func (db *Database) GenerateFts(since int64) error {
	stmt, err := db.conn.Prepare(sql_generate_fts)

	if err != nil {
		return err
	}

	stmt.Exec(since)

	return nil
}

// Performs a query upon the database where the only arguments are the page range.
// This is useful for thing such as popular and recent posts.
func (db *Database) PaginatedQuery(query string, page int) ([]*Post, error) {
	page_size := 25
	posts := make([]*Post, 0, page_size)

	rows, err := db.conn.Query(query, page_size*page,
		page_size)

	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var post Post

		err := rows.Scan(&post.Id, &post.InfoHash, &post.Title, &post.Size,
			&post.FileCount, &post.Seeders, &post.Leechers, &post.UploadDate,
			&post.Tags, &post.Meta)

		if err != nil {
			return nil, err
		}

		posts = append(posts, &post)
	}

	return posts, nil
}

// Returns a page of posts ordered by upload data, descending.
func (db *Database) QueryRecent(page int) ([]*Post, error) {
	return db.PaginatedQuery(sql_query_recent_post, page)
}

// Returns a page of posts ordered by popularity, descending.
// Popularity is a combination of seeders and leechers, weighted ever so slightly
// towards seeders.
func (db *Database) QueryPopular(page int) ([]*Post, error) {
	return db.PaginatedQuery(sql_query_popular_post, page)
}

// Perform a query on the FTS table. The results returned are used to pull actual
// results out of the post table, and these are returned.
func (db *Database) Search(query string, page, pageSize int) ([]*Post, error) {
	posts := make([]*Post, 0, pageSize)
	rows, err := db.conn.Query(sql_search_post, query, page*pageSize,
		pageSize)

	if err != nil {
		return nil, err
	}

	for rows.Next() {

		var result uint

		err = rows.Scan(&result)

		if err != nil {
			return nil, err
		}

		post, err := db.QueryPostId(result)

		if err != nil {
			return nil, err
		}

		posts = append(posts, &post)
	}

	return posts, nil
}

// Return a single post given it's id.
func (db *Database) QueryPostId(id uint) (Post, error) {
	var post Post
	rows, err := db.conn.Query(sql_query_post_id, id)

	if err != nil {
		return post, err
	}

	for rows.Next() {

		err := rows.Scan(&post.Id, &post.InfoHash, &post.Title, &post.Size,
			&post.FileCount, &post.Seeders, &post.Leechers, &post.UploadDate,
			&post.Tags, &post.Meta)

		if err != nil {
			return post, err
		}
	}

	return post, nil
}

// Return a single piece given it's id. Optionally store the posts as well,
// otherwise we just get a hash.
func (db *Database) QueryPiece(id uint, store bool) (*Piece, error) {
	page_size := PieceSize // TODO: Configure this elsewhere
	var piece Piece
	piece.Setup()
	piece.Id = id

	rows, err := db.conn.Query(sql_query_paged_post, id*uint(page_size),
		page_size)

	if err != nil {
		return nil, err
	}

	for rows.Next() {

		var post Post

		err := rows.Scan(&post.Id, &post.InfoHash, &post.Title, &post.Size,
			&post.FileCount, &post.Seeders, &post.Leechers, &post.UploadDate,
			&post.Tags, &post.Meta)

		if err != nil {
			return nil, err
		}

		piece.Add(post, store)
	}

	return &piece, nil
}

// Very simmilar to QueryPiece, except this returns a channel and streams posts
// out as they arrive. Queries a range of posts, so you can ask for 100 posts
// starting at an id.
func (db *Database) QueryPiecePosts(start, length int, store bool) chan *Post {
	ret := make(chan *Post)
	page_size := PieceSize // TODO: Configure this elsewhere

	go func() {
		defer close(ret)

		rows, err := db.conn.Query(sql_query_paged_post, start*page_size,
			start+page_size*length)

		if err != nil {
			return
		}

		for rows.Next() {

			var post Post

			err := rows.Scan(&post.Id, &post.InfoHash, &post.Title, &post.Size,
				&post.FileCount, &post.Seeders, &post.Leechers, &post.UploadDate,
				&post.Tags, &post.Meta)

			if err != nil {
				log.Error(err)
				return
			}

			ret <- &post
		}

	}()

	return ret
}

// How many posts are in the database?
func (db *Database) PostCount() uint {
	var res uint

	db.conn.QueryRow(sql_count_post).Scan(&res)

	return res
}

// Add a metadata key/value.
func (db *Database) AddMeta(pid int, value string) error {

	stmt, err := db.conn.Prepare(sql_attach_meta)

	if err != nil {
		return err
	}

	_, err = stmt.Exec(value, pid)

	if err != nil {
		return err
	}

	return nil
}

func (db *Database) Suggest(query string) ([]string, error) {
	suggest_size := 5

	ret := make([]string, 0, suggest_size)
	rows, err := db.conn.Query(sql_suggest_posts, query, suggest_size)

	if err != nil {
		return nil, err
	}

	for rows.Next() {

		var result string

		err = rows.Scan(&result)

		if err != nil {
			return nil, err
		}

		ret = append(ret, result)
	}

	return ret, nil
}

// Close the database connection.
func (db *Database) Close() {
	db.conn.Close()
}

package data

const sql_create_post_table string = `CREATE TABLE IF NOT EXISTS 
										post(
											id INTEGER PRIMARY KEY NOT NULL,
											info_hash STRING UNIQUE,
											title STRING NOT NULL,
											size INTEGER NOT NULL,
											file_count INTEGER NOT NULL,
											seeders INTEGER NOT NULL,
											leechers INTEGER NOT NULL,
											upload_date INTEGER NOT NULL,
											tags STRING,
											meta STRING
										)`

const sql_create_fts_post string = `CREATE VIRTUAL TABLE IF NOT EXISTS
									fts_post using fts4(
										content="post",
										title,
										seeders,
										leechers
									)`

const sql_create_upload_date_index string = `CREATE INDEX IF NOT EXISTS
											port_upload_date_index
											ON post(upload_date)`

const sql_insert_post string = `INSERT OR IGNORE INTO post(
									info_hash,
									title,
									size,
									file_count,
									seeders,
									leechers,
									upload_date,
									tags,
									meta
								) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`

const sql_attach_meta string = `UPDATE POST
								SET meta=?
								WHERE id=?`

const sql_generate_fts string = `INSERT OR IGNORE INTO fts_post(
								docid,
								title,
								seeders,
								leechers)
							SELECT id, title, seeders, leechers FROM post 
							WHERE id >= ?`

const sql_query_recent_post string = `SELECT 	 * FROM post
												 ORDER BY upload_date DESC
												 LIMIT ?,?`

const sql_query_popular_post string = ` SELECT * FROM(
													SELECT * FROM post 
													ORDER BY upload_date DESC
													LIMIT 10000
												)
												 ORDER BY seeders + leechers DESC
												 LIMIT ?,?`

const sql_query_post_id string = `SELECT 	 * FROM post
												 WHERE id = ?`

const sql_query_paged_post string = `SELECT 	 * FROM post
												 WHERE id > ?
												 LIMIT 0,?`

// Seeders are weighted, things with more seeders are better than things with
// more leechers, though both are important.
// (for one, seeders DO still upload, and are indicative of popularity)
const sql_search_post string = `SELECT docid FROM fts_post
									WHERE title MATCH ?
									ORDER BY ((seeders * 1.1) + leechers) DESC
									LIMIT ?,?`

const sql_suggest_posts string = `SELECT title FROM (
										SELECT * FROM post
										ORDER BY upload_date DESC
										LIMIT 100000
									)
									WHERE title LIKE ?
									ORDER BY (seeders * 1.1) + leechers DESC
									LIMIT 0,?`

const sql_count_post = `SELECT MAX(id) FROM post`

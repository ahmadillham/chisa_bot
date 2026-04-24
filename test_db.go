package main
import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"google.golang.org/protobuf/proto"
)
func main() {
	db, err := sql.Open("sqlite3", "file:bot.db?_journal_mode=WAL&_foreign_keys=on")
	if err != nil { panic(err) }
	rows, err := db.Query("SELECT stanza_id, protobuf, created_at FROM message_cache ORDER BY created_at DESC LIMIT 5")
	if err != nil { panic(err) }
	for rows.Next() {
		var id string
		var b []byte
		var ts int64
		rows.Scan(&id, &b, &ts)
		msg := &waProto.Message{}
		proto.Unmarshal(b, msg)
		fmt.Printf("ID: %s | Time: %d | Type: %T\n", id, ts, msg)
	}
}

package main

import (
	_ "github.com/mattn/go-sqlite3"
	"github.com/jmoiron/sqlx"

	"github.com/thoj/go-ircevent"

	"fmt"
	"log"
	"crypto/tls"
	"strings"
	"time"
)

type Tell struct {
	Id int
	Sender string
	Target string
	Message string
	TimeSent int `db:"time_sent"`
	IsRead int `db:"is_read"`
}

const channel = "#Téladiaire"
const serverssl = "irc.inframonde.org:6697"

func insert_tell(db *sqlx.DB, e *irc.Event) bool {
	data := strings.Split(e.Message(), " ")
	sender := e.Nick
	is_read := 0
	time_sent := time.Now().Unix()

	if data[0] == ".tell" && len(data) > 2{
		target := data[1]
		message := strings.Join(data[2:], " ")
		fmt.Printf("[%s to %s : %s] %d", sender, target, message, is_read)

		tx := db.MustBegin()
		tx.MustExec("INSERT INTO tell (sender, target, message, time_sent, is_read) VALUES ($1, $2, $3, $4, $5)", sender, target, message, time_sent, is_read)
		tx.Commit()

		return true
	} else {
		return false
	}
}

func search_tells(db *sqlx.DB, nickname string) (int, []Tell) {
	messages := []Tell{}
	db.Select(&messages, "SELECT * FROM tell WHERE target=$1 AND is_read=0", nickname)
	count := len(messages)

	return count, messages
}

func mark_as_read(db *sqlx.DB, nickname string) {
	tx := db.MustBegin()
	tx.MustExec("UPDATE tell SET is_read = 1 WHERE target=$1", nickname)
	tx.Commit()
}

func main() {
	db, err := sqlx.Connect("sqlite3", "_dummy.db")
	if err != nil {
		log.Fatalln(err)
	}

	irc_nick := "Guillaume"
	irc_conn := irc.IRC(irc_nick, "Guillaume")
	irc_conn.VerboseCallbackHandler = true
	irc_conn.Debug = true
	irc_conn.UseTLS = true
	irc_conn.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	irc_conn.AddCallback("001", func(e *irc.Event) { irc_conn.Join(channel)})
	irc_conn.AddCallback("336", func(e *irc.Event) { })
	irc_conn.AddCallback("JOIN", func (e *irc.Event) {
		if i, tells := search_tells(db, e.Nick); i > 0 {
			for _, v := range(tells) {
				irc_conn.Noticef(v.Target, "%s said : [%s]", v.Sender, v.Message)
			}

			mark_as_read(db, e.Nick)
		}
		})
	irc_conn.AddCallback("PRIVMSG", func (e *irc.Event) {
		if insert_tell(db, e) {
			irc_conn.Notice(e.Nick, "Done.")
		} else {
			irc_conn.Notice(e.Nick, "Error.")
		}

		if i, tells := search_tells(db, e.Nick); i > 0 {
			for _, v := range(tells) {
				loc, _ := time.LoadLocation("Europe/Paris")
				tm := time.Unix(int64(v.TimeSent), 0).In(loc).Format("15:04 - 02/01")
				irc_conn.Noticef(v.Target, "[%s] %s> %s", tm, v.Sender, v.Message)
			}

			mark_as_read(db, e.Nick)
		}
		})

	err = irc_conn.Connect(serverssl)
	if err != nil {
		log.Fatalln(err)
	}

	irc_conn.Loop()

}
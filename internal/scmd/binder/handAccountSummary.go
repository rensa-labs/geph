package binder

import (
	"encoding/base32"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"gopkg.in/bunsim/natrium.v1"
)

func (cmd *Command) handAccountSummary(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("content-type", "application/json")
	var req struct {
		PrivKey []byte
	}
	defer r.Body.Close()
	// populate from json
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Println("handAccountSummary:", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if len(req.PrivKey) != natrium.ECDHKeyLength {
		log.Println("handAccountSummary: given ECDH key malformed")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// get user id
	uid := strings.ToLower(
		base32.StdEncoding.EncodeToString(
			natrium.SecureHash(natrium.ECDHPrivate(req.PrivKey).PublicKey(), nil)[:10]))
	// set up response
	var resp struct {
		Username string
		RegDate  string
		Balance  int
	}
	var regdate time.Time
	// query the database now
	tx, err := cmd.pgdb.Begin()
	if err != nil {
		log.Println("handAccountSummary:", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	_, err = tx.Exec("SET TRANSACTION ISOLATION LEVEL SERIALIZABLE")
	if err != nil {
		log.Println("handAccountSummary:", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = tx.QueryRow(`SELECT Uname, Ctime, Mbs
		FROM AccInfo, AccBalances
		WHERE AccInfo.Uid = AccBalances.Uid
		AND AccInfo.Uid = $1`, uid).Scan(&resp.Username, &regdate, &resp.Balance)
	if err != nil {
		log.Println("handAccountSummary:", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	tx.Commit()
	resp.RegDate = regdate.Format(time.RFC3339)
	// write back response
	j, _ := json.MarshalIndent(&resp, "", "  ")
	w.Write(j)
}

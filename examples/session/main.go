package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/iamlongalong/diskv/diskv"
	"github.com/iamlongalong/diskv/gkv"
)

var sessionStore *gkv.Gkv[User]

func init() {
	db, err := diskv.CreateDB(context.Background(), &diskv.CreateConfig{
		Dir:     "./test/.data",
		KeysLen: 500,
		MaxLen:  128,
	})
	if err != nil {
		log.Fatal(err)
	}

	sessionStore = gkv.New[User](db)
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		user := &User{}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		defer r.Body.Close()

		err = json.Unmarshal(body, user)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		sessionID := newID()
		user.SessionID = sessionID
		user.LoginAt = time.Now()
		user.Expire = 1 * time.Hour
		user.LastActive = time.Now()

		err = sessionStore.Set(r.Context(), sessionID, user)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		res, _ := json.Marshal(map[string]any{
			"session_id": user.SessionID,
			"user":       user,
		})

		w.WriteHeader(http.StatusOK)
		w.Write(res)
	})

	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		sessionID := query.Get("session_id")
		if sessionID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized, no session_id in query"))
			return
		}

		// 从带类型的 store 中获取数据，得到的是 User 类型的实例
		user, ok, err := sessionStore.Get(r.Context(), sessionID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized, login first"))
			return
		}

		user.LastActive = time.Now()
		sessionStore.Set(r.Context(), sessionID, user) // 重新塞回去

		userbs, _ := json.Marshal(map[string]any{"user": user})
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(userbs))
	})

	mux.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		sessionID := query.Get("session_id")
		if sessionID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized, no session_id in query"))
			return
		}

		err := sessionStore.Set(r.Context(), sessionID, nil)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	fmt.Println("server listening on 127.0.0.1:8080")
	if err := http.ListenAndServe("127.0.0.1:8080", mux); err != nil {
		log.Fatal(err)
	}
}

type User struct {
	SessionID string `json:"session_id"`

	UserID    string `json:"user_id"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Slogan    string `json:"slogan"`

	LoginAt    time.Time
	LastActive time.Time // 最近一次活跃时间
	Expire     time.Duration

	Ext map[string]any `json:"ext"`
}

const seeds = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func newID() string {
	id := make([]byte, 10)
	for i, _ := range id {
		id[i] = seeds[rand.Intn(len(seeds))]
	}
	return string(id)
}

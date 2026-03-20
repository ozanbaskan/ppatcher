package main

import (
	"context"
	"net/http"
)

type contextKey string

const userContextKey contextKey = "user"

func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session")
		userID, ok := session.Values["user_id"].(int64)
		if !ok || userID == 0 {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		var user User
		err := db.QueryRow(r.Context(),
			`SELECT id, google_id, email, name, avatar_url, created_at, updated_at
			 FROM users WHERE id = $1`, userID,
		).Scan(&user.ID, &user.GoogleID, &user.Email, &user.Name, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			session.Options.MaxAge = -1
			session.Save(r, w)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, &user)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func currentUser(r *http.Request) *User {
	u, _ := r.Context().Value(userContextKey).(*User)
	return u
}

func optionalUser(r *http.Request) *User {
	session, _ := store.Get(r, "session")
	userID, ok := session.Values["user_id"].(int64)
	if !ok || userID == 0 {
		return nil
	}

	var user User
	err := db.QueryRow(r.Context(),
		`SELECT id, google_id, email, name, avatar_url, created_at, updated_at
		 FROM users WHERE id = $1`, userID,
	).Scan(&user.ID, &user.GoogleID, &user.Email, &user.Name, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil
	}
	return &user
}

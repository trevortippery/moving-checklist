package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

type Envelope map[string]interface{}

func WriteJSON(w http.ResponseWriter, status int, data Envelope) error {
	js, err := json.MarshalIndent(data, "", "")
	if err != nil {
		return err
	}

	js = append(js, '\n')
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)
	return nil
}

func ReadIDParam(r *http.Request) (int64, error) {
	idParam := chi.URLParam(r, "id")
	if idParam == "" {
		return 0, errors.New("invalid id parameter")
	}

	// Log the idParam to see the URL parameter
	fmt.Println("Extracted ID Param:", idParam)

	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return 0, errors.New("invalid id parameter type")
	}

	// Log the parsed ID to confirm it's valid
	fmt.Println("Parsed ID:", id)

	return id, nil
}

func HashPassword(password []byte) ([]byte, error) {
	return bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
}

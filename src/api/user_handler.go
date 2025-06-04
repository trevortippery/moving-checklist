package api

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/trevortippery/moving-checklist/db"
	"github.com/trevortippery/moving-checklist/middleware"
	"github.com/trevortippery/moving-checklist/utils"
)

var emailRegex = regexp.MustCompile(`^[\w\.-]+@[\w\.-]+\.\w{2,}$`)

type UserHandler struct {
	userStore  db.UserStore
	tokenStore db.TokenStore
	logger     *log.Logger
}

type UserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func NewUserHandler(userStore db.UserStore, tokenStore db.TokenStore, logger *log.Logger) *UserHandler {
	return &UserHandler{
		userStore:  userStore,
		tokenStore: tokenStore,
		logger:     logger,
	}
}

func (uh *UserHandler) HandleRegisterUser(w http.ResponseWriter, r *http.Request) {
	var input UserRequest
	const funcName = "HandleRegisterUser"

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		uh.logger.Printf("Error in %s: Decoding Register User - %v", funcName, err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"error": "invalid request body"})
		return
	}

	validationErrors := validateUserInput(input, ValidateCreate)
	if len(validationErrors) > 0 {
		uh.logger.Printf("Error in %s: Validating input - %+v", funcName, validationErrors)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"errors": validationErrors})
		return
	}

	hashedPassword, err := utils.HashPassword([]byte(input.Password))
	if err != nil {
		uh.logger.Printf("Error in %s: Hashing password - %v", funcName, err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"error": "something went wrong"})
		return
	}

	user := db.User{
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: string(hashedPassword),
	}

	createdUser, err := uh.userStore.RegisterUser(r.Context(), &user)
	if err != nil {
		uh.logger.Printf("Error in %s: Register user - %v", funcName, err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"error": "failed to create user"})
		return
	}

	token, err := uh.tokenStore.GenerateToken(r.Context(), int64(createdUser.ID), 24*time.Hour, "auth")
	if err != nil {
		uh.logger.Printf("Error in %s: Generating token - %v", funcName, err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"error": "failed to generate token"})
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"user": map[string]interface{}{
			"id":         createdUser.ID,
			"username":   createdUser.Username,
			"email":      createdUser.Email,
			"created_at": createdUser.CreatedAt,
			"updated_at": createdUser.UpdatedAt,
			"token":      token,
		},
	})
}

func (uh *UserHandler) HandleDeleteUser(w http.ResponseWriter, r *http.Request) {
	const funcName = "HandleDeleteUser"

	user := middleware.GetUser(r)
	if user == nil {
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "not authenticated"})
		return
	}

	err := uh.userStore.DeleteUser(r.Context(), int64(user.ID))
	if err != nil {
		uh.logger.Printf("Error in %s: Delete user - %v", funcName, err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"error": "failed to delete user"})
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"message": "user deleted successfully",
	})
}

func (uh *UserHandler) HandleUpdateUser(w http.ResponseWriter, r *http.Request) {
	const funcName = "HandleUpdateUser"

	user := middleware.GetUser(r)
	if user == nil {
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "not authenticated"})
		return
	}

	var input UserRequest
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		uh.logger.Printf("Error in %s: Decoding input - %v", funcName, err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"error": "invalid request body"})
		return
	}

	validationErrors := validateUserInput(input, ValidateUpdate)
	if len(validationErrors) > 0 {
		uh.logger.Printf("Error in %s: Validation - %+v", funcName, validationErrors)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"errors": validationErrors})
		return
	}

	// Update allowed fields
	user.Username = input.Username
	user.Email = input.Email

	// Only hash password if it's being updated
	if input.Password != "" {
		hashedPassword, err := utils.HashPassword([]byte(input.Password))
		if err != nil {
			uh.logger.Printf("Error in %s: Hashing password - %v", funcName, err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"error": "failed to update user"})
			return
		}
		user.PasswordHash = string(hashedPassword)
	}

	err = uh.userStore.UpdateUser(r.Context(), user)
	if err != nil {
		uh.logger.Printf("Error in %s: Updating user - %v", funcName, err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"error": "failed to update user"})
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"user": map[string]interface{}{
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"updated_at": time.Now().UTC(),
		},
	})
}

func validateUserInput(input UserRequest, mode ValidationMode) map[string]string {
	errors := make(map[string]string)

	if mode == ValidateCreate {
		if strings.TrimSpace(input.Username) == "" {
			errors["username"] = "username is required"
		} else if len(input.Username) > 50 {
			errors["username"] = "username must be less than 50 characters"
		}

		if strings.TrimSpace(input.Password) == "" {
			errors["password"] = "password is required"
		} else if len(input.Password) < 8 {
			errors["password"] = "password must be at least 8 characters"
		}
	}

	return errors
}

package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

type userTestCase struct {
	name    string
	user    *User
	wantErr bool
}

func TestCreateUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewPostgresUserStore(db)
	ctx := context.Background()

	tests := []userTestCase{
		{
			name:    "Valid user",
			user:    validUser("example", "example@example.com"),
			wantErr: false,
		},
		{
			name:    "Invalid user with same username",
			user:    validUser("example", "example2@example.com"),
			wantErr: true,
		},
		{
			name:    "Invalid user with same email",
			user:    validUser("another", "example@example.com"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.user == nil {
				t.Skip("nil user not supported in this test")
			}

			createdUser, err := store.RegisterUser(ctx, tt.user)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, createdUser)
			} else {
				require.NoError(t, err)
				require.NotNil(t, createdUser)

				assert.Equal(t, tt.user.Username, createdUser.Username)
				assert.Equal(t, tt.user.Email, createdUser.Email)
				assert.Greater(t, createdUser.ID, int(0), "ID should be set")
				assert.False(t, createdUser.CreatedAt.IsZero(), "CreatedAt should be set")
				assert.False(t, createdUser.UpdatedAt.IsZero(), "UpdatedAt should be set")
			}
		})
	}

}

func validUser(username string, email string) *User {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("testpass123"), bcrypt.DefaultCost)
	return &User{
		Username: username,
		Email:    email,
		Password: hashedPassword,
	}
}

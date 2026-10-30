package repository

import (
	"sync"
	"testing"

	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/schema"
)

func TestUserModel_GORMColumnMapping(t *testing.T) {
	namer := schema.NamingStrategy{}
	s, err := schema.Parse(&models.UserModel{}, &sync.Map{}, namer)
	require.NoError(t, err)

	assert.Equal(t, "users", s.Table)

	fieldDBNames := make(map[string]string)
	for _, f := range s.Fields {
		fieldDBNames[f.Name] = f.DBName
	}

	assert.Equal(t, "id", fieldDBNames["ID"])
	assert.Equal(t, "full_name", fieldDBNames["FullName"])
	assert.Equal(t, "email", fieldDBNames["Email"])
	assert.Equal(t, "phone", fieldDBNames["Phone"])
	assert.Equal(t, "password_hash", fieldDBNames["PasswordHash"])
	assert.Equal(t, "role", fieldDBNames["Role"])
	assert.Equal(t, "status", fieldDBNames["Status"])
	assert.Equal(t, "profile_image", fieldDBNames["ProfileImage"])
	assert.Equal(t, "created_at", fieldDBNames["CreatedAt"])
	assert.Equal(t, "updated_at", fieldDBNames["UpdatedAt"])
}

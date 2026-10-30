package repository

import (
	"sync"
	"testing"

	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/schema"
)

func TestOrganizerModel_GORMColumnMapping(t *testing.T) {
	namer := schema.NamingStrategy{}
	s, err := schema.Parse(&models.OrganizerModel{}, &sync.Map{}, namer)
	require.NoError(t, err)

	assert.Equal(t, "organizers", s.Table)

	fieldDBNames := make(map[string]string)
	for _, f := range s.Fields {
		fieldDBNames[f.Name] = f.DBName
	}

	assert.Equal(t, "id", fieldDBNames["ID"])
	assert.Equal(t, "user_id", fieldDBNames["UserID"])
	assert.Equal(t, "status", fieldDBNames["Status"])
	assert.Equal(t, "created_at", fieldDBNames["CreatedAt"])
	assert.Equal(t, "updated_at", fieldDBNames["UpdatedAt"])
}

func TestOrganizerProfileModel_GORMColumnMapping(t *testing.T) {
	namer := schema.NamingStrategy{}
	s, err := schema.Parse(&models.OrganizerProfileModel{}, &sync.Map{}, namer)
	require.NoError(t, err)

	assert.Equal(t, "organizer_profiles", s.Table)

	fieldDBNames := make(map[string]string)
	for _, f := range s.Fields {
		fieldDBNames[f.Name] = f.DBName
	}

	assert.Equal(t, "id", fieldDBNames["ID"])
	assert.Equal(t, "organizer_id", fieldDBNames["OrganizerID"])
	assert.Equal(t, "business_name", fieldDBNames["BusinessName"])
	assert.Equal(t, "business_type", fieldDBNames["BusinessType"])
	assert.Equal(t, "description", fieldDBNames["Description"])
	assert.Equal(t, "phone", fieldDBNames["Phone"])
	assert.Equal(t, "email", fieldDBNames["Email"])
	assert.Equal(t, "website", fieldDBNames["Website"])
	assert.Equal(t, "logo_url", fieldDBNames["LogoURL"])
	assert.Equal(t, "gst_number", fieldDBNames["GSTNumber"])
	assert.Equal(t, "pan_number", fieldDBNames["PANNumber"])
}

package httpapi

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"theblock/site/internal/auth"
	"theblock/site/internal/repo"
)

func TestCanEditContentOwnership(t *testing.T) {
	own := repo.Content{AuthorID: 7}
	other := repo.Content{AuthorID: 8}

	assert.True(t, canEditContent(auth.User{ID: 7, Role: "Yazar"}, own))
	assert.False(t, canEditContent(auth.User{ID: 7, Role: "Yazar"}, other))
	assert.True(t, canEditContent(auth.User{ID: 7, Role: "Muhabir"}, own))
	assert.False(t, canEditContent(auth.User{ID: 7, Role: "Muhabir"}, other))
	assert.True(t, canEditContent(auth.User{ID: 9, Role: "Editör"}, other))
	assert.True(t, canEditContent(auth.User{ID: 9, Role: "Yönetici"}, other))
	assert.False(t, canEditContent(auth.User{ID: 7, Role: "Üye"}, own))
	assert.False(t, canEditContent(auth.User{}, own))
}

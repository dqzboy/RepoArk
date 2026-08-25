package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"git-backup-web/server/internal/db"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestProfileResponseMasksToken(t *testing.T) {
	database := newConfigHandlerTestDB(t)
	profile := db.Profile{Platform: db.PlatformGitCode, GitToken: "secret-token", ServerName: "blog-prod"}
	if err := database.Create(&profile).Error; err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Params = gin.Params{{Key: "platform", Value: db.PlatformGitCode}}
	GetProfile(database)(context)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["git_token"] != "********" || body["token_configured"] != true {
		t.Fatalf("masked token response = %#v", body)
	}
	if strings.Contains(recorder.Body.String(), "secret-token") {
		t.Fatal("profile response leaked token")
	}
}

func TestGetProfileTokenRequiresAdmin(t *testing.T) {
	database := newConfigHandlerTestDB(t)
	for _, user := range []db.User{
		{Username: "admin", Role: "admin"},
		{Username: "viewer", Role: "viewer"},
	} {
		if err := database.Create(&user).Error; err != nil {
			t.Fatal(err)
		}
	}
	profile := db.Profile{Platform: db.PlatformGitCode, GitToken: "secret-token", ServerName: "blog-prod"}
	if err := database.Create(&profile).Error; err != nil {
		t.Fatal(err)
	}

	t.Run("admin can reveal", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(recorder)
		context.Params = gin.Params{{Key: "platform", Value: db.PlatformGitCode}}
		context.Set("username", "admin")
		GetProfileToken(database)(context)
		if recorder.Code != http.StatusOK || recorder.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("status = %d, cache-control = %q, body = %s", recorder.Code, recorder.Header().Get("Cache-Control"), recorder.Body.String())
		}
		var body map[string]string
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body["git_token"] != "secret-token" {
			t.Fatalf("git_token = %q", body["git_token"])
		}
	})

	t.Run("viewer is forbidden", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(recorder)
		context.Params = gin.Params{{Key: "platform", Value: db.PlatformGitCode}}
		context.Set("username", "viewer")
		GetProfileToken(database)(context)
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
		}
	})
}

func newConfigHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	gin.SetMode(gin.TestMode)
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "handler.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&db.Profile{}, &db.User{}); err != nil {
		t.Fatal(err)
	}
	return database
}

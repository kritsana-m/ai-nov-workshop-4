package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"kbtg-ai-workshop-nov/workshop-4/backend/internal/models"
	"kbtg-ai-workshop-nov/workshop-4/backend/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupRouterWithDB(t *testing.T) *gin.Engine {
	t.Helper()
	// Use a per-test in-memory database to avoid cross-test interference
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := store.InitDB(dsn)
	require.NoError(t, err)
	store.SetDB(db)
	r := gin.New()
	RegisterRoutes(r)
	return r
}

func httpDo(r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var req *http.Request
	if body != nil {
		b, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestUserCRUD(t *testing.T) {
	r := setupRouterWithDB(t)

	// Create user A
	aReq := models.User{MemberCode: "A001", Name: "Alice", RemainingPoints: 100}
	w := httpDo(r, "POST", "/users", aReq)
	require.Equal(t, http.StatusCreated, w.Code)
	var a models.User
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &a))
	require.NotZero(t, a.ID)
	require.Equal(t, "A001", a.MemberCode)

	// Create user B
	bReq := models.User{MemberCode: "B001", Name: "Bob", RemainingPoints: 10}
	w = httpDo(r, "POST", "/users", bReq)
	require.Equal(t, http.StatusCreated, w.Code)
	var b models.User
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &b))

	// List users (should return both)
	w = httpDo(r, "GET", "/users", nil)
	require.Equal(t, http.StatusOK, w.Code)
	var users []models.User
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &users))
	require.GreaterOrEqual(t, len(users), 2)

	// Get user A by id
	w = httpDo(r, "GET", "/users/"+strconv.FormatUint(uint64(a.ID), 10), nil)
	require.Equal(t, http.StatusOK, w.Code)
	var gotA models.User
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &gotA))
	require.Equal(t, a.MemberCode, gotA.MemberCode)

	// Update user A
	a.Name = "Alice Updated"
	w = httpDo(r, "PUT", "/users/"+strconv.FormatUint(uint64(a.ID), 10), a)
	require.Equal(t, http.StatusOK, w.Code)
	var upA models.User
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &upA))
	require.Equal(t, "Alice Updated", upA.Name)

	// Delete user B
	w = httpDo(r, "DELETE", "/users/"+strconv.FormatUint(uint64(b.ID), 10), nil)
	require.Equal(t, http.StatusNoContent, w.Code)

	// Getting deleted user should return 404
	w = httpDo(r, "GET", "/users/"+strconv.FormatUint(uint64(b.ID), 10), nil)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestTransferSuccessAndLedgers(t *testing.T) {
	r := setupRouterWithDB(t)
	db := store.GetDB()

	// Create from and to users
	from := models.User{MemberCode: "F1", Name: "From", RemainingPoints: 100}
	to := models.User{MemberCode: "T1", Name: "To", RemainingPoints: 5}
	require.NoError(t, db.Create(&from).Error)
	require.NoError(t, db.Create(&to).Error)

	// Transfer 30 points
	trReq := map[string]interface{}{"fromUserId": from.ID, "toUserId": to.ID, "amount": 30, "note": "thanks"}
	w := httpDo(r, "POST", "/transfers", trReq)
	require.Equal(t, http.StatusCreated, w.Code)
	var resp struct {
		Transfer models.Transfer `json:"transfer"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	tr := resp.Transfer
	require.Equal(t, 30, tr.Amount)
	require.Equal(t, "completed", tr.Status)

	// Check balances updated
	var f models.User
	var tuser models.User
	require.NoError(t, db.First(&f, from.ID).Error)
	require.NoError(t, db.First(&tuser, to.ID).Error)
	require.Equal(t, 70, f.RemainingPoints)
	require.Equal(t, 35, tuser.RemainingPoints)

	// Check point ledgers (2 entries)
	var ledgers []models.PointLedger
	require.NoError(t, db.Where("transfer_id = ?", tr.ID).Find(&ledgers).Error)
	require.Len(t, ledgers, 2)
}

func TestTransferInsufficientPoints(t *testing.T) {
	r := setupRouterWithDB(t)
	db := store.GetDB()

	from := models.User{MemberCode: "F2", Name: "Low", RemainingPoints: 5}
	to := models.User{MemberCode: "T2", Name: "Receiver", RemainingPoints: 0}
	require.NoError(t, db.Create(&from).Error)
	require.NoError(t, db.Create(&to).Error)

	trReq := map[string]interface{}{"fromUserId": from.ID, "toUserId": to.ID, "amount": 20}
	w := httpDo(r, "POST", "/transfers", trReq)
	require.Equal(t, http.StatusConflict, w.Code)
	var m map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &m))
	require.Equal(t, "insufficient points", m["error"])

	// Ensure no transfer created
	var count int64
	require.NoError(t, db.Model(&models.Transfer{}).Count(&count).Error)
	require.Equal(t, int64(0), count)
}

func TestTransferInvalidUserAndSelfTransfer(t *testing.T) {
	r := setupRouterWithDB(t)
	db := store.GetDB()

	// Non-existent from user
	trReq := map[string]interface{}{"fromUserId": 9999, "toUserId": 1, "amount": 1}
	w := httpDo(r, "POST", "/transfers", trReq)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	// Create a user and attempt transfer to self
	u := models.User{MemberCode: "S1", Name: "Self", RemainingPoints: 50}
	require.NoError(t, db.Create(&u).Error)
	trReq2 := map[string]interface{}{"fromUserId": u.ID, "toUserId": u.ID, "amount": 1}
	w = httpDo(r, "POST", "/transfers", trReq2)
	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
	var m map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &m))
	require.Equal(t, "cannot transfer to self", m["error"])
}

func TestListTransfersFilteringAndPagination(t *testing.T) {
	r := setupRouterWithDB(t)
	db := store.GetDB()

	// Create users
	u1 := models.User{MemberCode: "L1", Name: "U1", RemainingPoints: 100}
	u2 := models.User{MemberCode: "L2", Name: "U2", RemainingPoints: 100}
	require.NoError(t, db.Create(&u1).Error)
	require.NoError(t, db.Create(&u2).Error)

	// Create multiple transfers via direct DB so timestamps exist
	for i := 0; i < 5; i++ {
		tr := models.Transfer{FromUserID: u1.ID, ToUserID: u2.ID, Amount: 1, Status: "completed", IdempotencyKey: fmt.Sprintf("%s-%d-%d", t.Name(), i, time.Now().UnixNano()), CreatedAt: time.Now(), UpdatedAt: time.Now()}
		require.NoError(t, db.Create(&tr).Error)
	}
	for i := 0; i < 3; i++ {
		tr := models.Transfer{FromUserID: u2.ID, ToUserID: u1.ID, Amount: 2, Status: "completed", IdempotencyKey: fmt.Sprintf("%s-%d-%d", t.Name(), i+10, time.Now().UnixNano()), CreatedAt: time.Now(), UpdatedAt: time.Now()}
		require.NoError(t, db.Create(&tr).Error)
	}

	// List transfers for u1
	w := httpDo(r, "GET", "/transfers?userId="+strconv.FormatUint(uint64(u1.ID), 10), nil)
	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data     []models.Transfer `json:"data"`
		Page     int               `json:"page"`
		PageSize int               `json:"pageSize"`
		Total    int64             `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.GreaterOrEqual(t, len(resp.Data), 1)
	require.GreaterOrEqual(t, resp.Total, int64(8))
}

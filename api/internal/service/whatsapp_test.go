package service_test
 
import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"
 
	"github.com/kilip/opus/api/ent"
	"github.com/kilip/opus/api/ent/wamessage"
	"github.com/kilip/opus/api/internal/model"
	"github.com/kilip/opus/api/internal/service"
	"github.com/kilip/opus/api/mocks"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	_ "github.com/mattn/go-sqlite3"
)
 
func TestConnect_DispatchesQR(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
 
	mockHub := mocks.NewMockSSEHub(ctrl)
	
	// Create a real in-memory store for testing
	container, err := sqlstore.New(context.Background(), "sqlite3", "file::memory:?cache=shared&_fk=1", nil)
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
 
	svc := service.NewWhatsAppService(nil, mockHub, nil, nil, container, slog.Default())
 
	// Expect QR dispatch
	mockHub.EXPECT().Publish(gomock.Any(), "user-1", gomock.Any()).Return(nil).AnyTimes()
 
	// Connect will start a background goroutine for QR. 
	// Since we are using an empty store, it WILL generate QR codes.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
 
	err = svc.Connect(ctx, "user-1")
	if err != nil {
		t.Errorf("expected no error from Connect, got %v", err)
	}
 
	// Wait a bit for the QR goroutine to run
	time.Sleep(500 * time.Millisecond)
}

func TestWhatsAppService_handleNewMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup in-memory Ent client
	db, err := ent.Open("sqlite3", "file::memory:?cache=shared&_fk=1")
	if err != nil {
		t.Fatalf("failed opening connection to sqlite: %v", err)
	}
	defer func() { _ = db.Close() }()
	if err := db.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed creating schema resources: %v", err)
	}

	mockRepo := mocks.NewMockWhatsAppRepository(ctrl)
	mockHub := mocks.NewMockSSEHub(ctrl)
	
	svc := service.NewWhatsAppService(mockRepo, mockHub, nil, db, nil, slog.Default())

	userID := "user-1"
	// Create user and session in DB
	u, _ := db.User.Create().SetID(userID).SetEmail("test@example.com").SetProvider("email").SetName("Test").Save(context.Background())
	sess, _ := db.WaSession.Create().SetID(userID).SetUserID(u.ID).SetStatus("CONNECTED").Save(context.Background())

	mockRepo.EXPECT().GetSessionByUserID(gomock.Any(), userID).Return(sess, nil)
	mockHub.EXPECT().Publish(gomock.Any(), userID, gomock.Any()).Return(nil)

	// Create a mock whatsmeow message event
	evt := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Sender:   types.NewJID("12345", types.DefaultUserServer),
				Chat:     types.NewJID("12345", types.DefaultUserServer),
				IsFromMe: false,
			},
			ID:        "MSG123",
			Timestamp: time.Now(),
		},
		Message: &waE2E.Message{
			Conversation: proto.String("Hello World"),
		},
	}

	// This should fail initially because handleNewMessage is not implemented or chat edge is required
	service.HandleNewMessage(svc, userID, evt)

	// Verify message saved in DB
	msg, err := db.WaMessage.Query().Where(wamessage.MessageIDEQ("MSG123")).Only(context.Background())
	if err != nil {
		t.Errorf("expected message to be saved, got error: %v", err)
	}
	if msg.Content != "Hello World" {
		t.Errorf("expected content 'Hello World', got '%s'", msg.Content)
	}
}

func TestWhatsAppService_handleHistorySync(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQueue := mocks.NewMockQueue(ctrl)
	svc := service.NewWhatsAppService(nil, nil, mockQueue, nil, nil, slog.Default())

	userID := "user-1"
	evt := &events.HistorySync{
		Data: &waHistorySync.HistorySync{
			SyncType: waHistorySync.HistorySync_INITIAL_BOOTSTRAP.Enum(),
		},
	}

	mockQueue.EXPECT().Push(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, job *model.Job) error {
		if job.Type != service.JobWhatsAppSyncHistory {
			t.Errorf("expected job type %s, got %s", service.JobWhatsAppSyncHistory, job.Type)
		}
		var payload struct {
			UserID string `json:"user_id"`
			Data   []byte `json:"data"`
		}
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			t.Errorf("failed to unmarshal job payload: %v", err)
		}
		if payload.UserID != userID {
			t.Errorf("expected user ID %s, got %s", userID, payload.UserID)
		}
		return nil
	})

	service.HandleHistorySync(svc, userID, evt)
}

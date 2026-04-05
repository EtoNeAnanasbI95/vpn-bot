package bot

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Server is the HTTP webhook server.
type Server struct {
	router      *Router
	webhookPath string
	listenAddr  string
}

func NewServer(router *Router, webhookPath, listenAddr string) *Server {
	return &Server{router: router, webhookPath: webhookPath, listenAddr: listenAddr}
}

// RegisterWebhook tells Telegram to POST updates to webhookURL+webhookPath.
func RegisterWebhook(bot *tgbotapi.BotAPI, webhookURL, webhookPath string) error {
	wh, err := tgbotapi.NewWebhook(webhookURL + webhookPath)
	if err != nil {
		return err
	}
	_, err = bot.Request(wh)
	return err
}

// Start begins listening for Telegram webhook updates. Blocks until ctx is done.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc(s.webhookPath, s.handleUpdate)

	srv := &http.Server{
		Addr:    s.listenAddr,
		Handler: mux,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("webhook server started", "addr", s.listenAddr, "path", s.webhookPath)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		return srv.Shutdown(context.Background())
	case err := <-errCh:
		return err
	}
}

func (s *Server) handleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("webhook: read body", "err", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var update tgbotapi.Update
	if err := json.Unmarshal(body, &update); err != nil {
		slog.Error("webhook: unmarshal update", "err", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Respond immediately — Telegram requires a fast 200 OK.
	// r.Context() is cancelled when the handler returns, so we detach.
	w.WriteHeader(http.StatusOK)

	go func() {
		defer func() {
			if p := recover(); p != nil {
				slog.Error("dispatch panic", "panic", p, "update_id", update.UpdateID)
			}
		}()
		s.router.Dispatch(context.Background(), update)
	}()
}

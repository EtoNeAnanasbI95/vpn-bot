package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"

	botpkg "github.com/EtoNeAnanasbI95/vpn-bot/internal/bot"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/logger"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/handler"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/session"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/config"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/guide"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/repository/sqlite"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/scheduler"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/usecase"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/xui"
)

func main() {
	// Load .env (ignore error if file doesn't exist — env may come from the shell).
	_ = godotenv.Load()

	logger.Setup(os.Getenv("DEBUG") == "true")

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}

	// ── Database ──────────────────────────────────────────────────────────────
	db, err := sqlite.Open(cfg.DBPath)
	if err != nil {
		slog.Error("open database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := sqlite.Migrate(db); err != nil {
		slog.Error("run migration", "err", err)
		os.Exit(1)
	}

	// ── Repositories ──────────────────────────────────────────────────────────
	userRepo := sqlite.NewUserRepository(db)
	paymentRepo := sqlite.NewPaymentRepository(db)
	connPayRepo := sqlite.NewConnectionPaymentRepository(db)
	connRequestRepo := sqlite.NewConnRequestRepository(db)

	// ── 3x-ui client ─────────────────────────────────────────────────────────
	var xuiClient xui.Client
	if cfg.XUIBaseURL != "" {
		xuiClient = xui.NewHTTPClient(cfg.XUIBaseURL, cfg.XUIUsername, cfg.XUIPassword)
		if err := xuiClient.Login(context.Background()); err != nil {
			slog.Warn("xui: initial login failed (will retry on demand)", "err", err)
		}
	}

	// ── Guide provider ────────────────────────────────────────────────────────
	guideProvider := guide.NewFSProvider(cfg.GuidesDir)

	// ── Use cases ─────────────────────────────────────────────────────────────
	userUC := usecase.NewUserUseCase(userRepo, cfg.AdminIDs)
	connUC := usecase.NewConnectionUseCase(xuiClient, cfg.XUIInboundID, cfg.XUIServerAddr, connPayRepo)
	paymentUC := usecase.NewPaymentUseCase(paymentRepo)
	guideUC := usecase.NewGuideUseCase(guideProvider)
	connRequestUC := usecase.NewConnRequestUseCase(connRequestRepo)

	useCases := &handler.UseCases{
		User:        userUC,
		Connection:  connUC,
		Payment:     paymentUC,
		Guide:       guideUC,
		ConnRequest: connRequestUC,
	}

	// ── Telegram bot ──────────────────────────────────────────────────────────
	bot, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		slog.Error("init telegram bot", "err", err)
		os.Exit(1)
	}
	slog.Info("bot authorised", "username", bot.Self.UserName)

	if cfg.RegisterWebhook {
		if err := botpkg.RegisterWebhook(bot, cfg.WebhookURL, cfg.WebhookPath); err != nil {
			slog.Error("register webhook", "err", err)
			os.Exit(1)
		}
		slog.Info("webhook registered", "url", cfg.WebhookURL+cfg.WebhookPath)
	} else {
		slog.Info("webhook registration skipped (REGISTER_WEBHOOK=false)")
	}

	// ── Session store ─────────────────────────────────────────────────────────
	sessions := session.NewMemoryStore()

	// ── Router & HTTP server ──────────────────────────────────────────────────
	router := botpkg.NewRouter(bot, cfg, useCases, sessions)
	server := botpkg.NewServer(router, cfg.WebhookPath, cfg.ListenAddr)

	// ── Payment reminder scheduler ────────────────────────────────────────────
	sched, err := scheduler.New(bot, userUC, paymentUC, connUC, cfg.ReminderTZ)
	if err != nil {
		slog.Error("init scheduler", "err", err)
		os.Exit(1)
	}
	sched.Start()
	defer sched.Stop()

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := server.Start(ctx); err != nil {
		slog.Error("server stopped", "err", err)
	}

	slog.Info("shutdown complete")
}

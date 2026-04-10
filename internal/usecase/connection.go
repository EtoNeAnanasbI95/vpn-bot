package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/domain"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/repository"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/xui"
)

type connectionUseCase struct {
	xuiClient     xui.Client
	xuiInboundID  int
	xuiServerAddr string
	connPayRepo   repository.ConnectionPaymentRepository
}

func NewConnectionUseCase(
	xuiClient xui.Client,
	xuiInboundID int,
	xuiServerAddr string,
	connPayRepo repository.ConnectionPaymentRepository,
) ConnectionUseCase {
	return &connectionUseCase{
		xuiClient:     xuiClient,
		xuiInboundID:  xuiInboundID,
		xuiServerAddr: xuiServerAddr,
		connPayRepo:   connPayRepo,
	}
}

func (uc *connectionUseCase) ListForUser(ctx context.Context, userID int64) ([]*domain.Connection, error) {
	inbound, err := uc.xuiClient.GetInbound(ctx, uc.xuiInboundID)
	if err != nil {
		return nil, fmt.Errorf("get inbound: %w", err)
	}

	var settings xui.InboundSettings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		return nil, fmt.Errorf("parse inbound settings: %w", err)
	}

	_, vlessBase, vlessErr := uc.inboundAndVLESSBase(inbound)
	if vlessErr != nil {
		slog.Warn("connection: could not build vless base, links will be empty", "err", vlessErr)
	}

	var conns []*domain.Connection
	for _, cl := range settings.Clients {
		if int64(cl.TgId) != userID {
			continue
		}
		link := ""
		if vlessErr == nil {
			link = buildVLESSLink(cl.ID, uc.xuiServerAddr, vlessBase, inbound.Port, cl.Comment)
		}
		conn := &domain.Connection{
			UUID:      cl.ID,
			UserID:    userID,
			Label:     cl.Comment,
			Link:      link,
			IsActive:  cl.Enable,
			PayStatus: domain.ConnPayFree,
		}
		if pay, err := uc.connPayRepo.GetByUUID(ctx, cl.ID); err == nil {
			conn.PayStatus = pay.Status
			conn.AdminID = pay.AdminID
		}
		conns = append(conns, conn)
	}
	return conns, nil
}

func (uc *connectionUseCase) GetByUUID(ctx context.Context, clientUUID string) (*domain.Connection, error) {
	inbound, err := uc.xuiClient.GetInbound(ctx, uc.xuiInboundID)
	if err != nil {
		return nil, fmt.Errorf("get inbound: %w", err)
	}

	var settings xui.InboundSettings
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		return nil, fmt.Errorf("parse inbound settings: %w", err)
	}

	_, vlessBase, vlessErr := uc.inboundAndVLESSBase(inbound)

	for _, cl := range settings.Clients {
		if cl.ID != clientUUID {
			continue
		}
		link := ""
		if vlessErr == nil {
			link = buildVLESSLink(cl.ID, uc.xuiServerAddr, vlessBase, inbound.Port, cl.Comment)
		}
		conn := &domain.Connection{
			UUID:      cl.ID,
			UserID:    int64(cl.TgId),
			Label:     cl.Comment,
			Link:      link,
			IsActive:  cl.Enable,
			PayStatus: domain.ConnPayFree,
		}
		if pay, err := uc.connPayRepo.GetByUUID(ctx, cl.ID); err == nil {
			conn.PayStatus = pay.Status
			conn.AdminID = pay.AdminID
		}
		return conn, nil
	}
	return nil, fmt.Errorf("client %s not found", clientUUID)
}

func (uc *connectionUseCase) GenerateQR(_ context.Context, link string) ([]byte, error) {
	png, err := qrcode.Encode(link, qrcode.Medium, 256)
	if err != nil {
		return nil, fmt.Errorf("generate qr: %w", err)
	}
	return png, nil
}

func (uc *connectionUseCase) Create(ctx context.Context, userID, adminID int64, tgTag, label string, isFree bool) (*domain.Connection, error) {
	if uc.xuiClient == nil {
		return nil, fmt.Errorf("3x-ui client is not configured")
	}
	if uc.xuiInboundID == 0 {
		return nil, fmt.Errorf("XUI_INBOUND_ID is not configured")
	}
	if uc.xuiServerAddr == "" {
		return nil, fmt.Errorf("XUI_SERVER_ADDR is not configured")
	}

	email := sanitizeEmail(fmt.Sprintf("%s-%s-%d-%s", tgTag, label, userID, uuid.New().String()))

	xuiCl, err := uc.xuiClient.CreateClient(ctx, uc.xuiInboundID, email, userID, label)
	if err != nil {
		return nil, fmt.Errorf("create xui client: %w", err)
	}

	fetchedInbound, err := uc.xuiClient.GetInbound(ctx, uc.xuiInboundID)
	if err != nil {
		return nil, fmt.Errorf("get inbound: %w", err)
	}
	inbound, vlessBase, err := uc.inboundAndVLESSBase(fetchedInbound)
	if err != nil {
		return nil, fmt.Errorf("build vless base: %w", err)
	}

	payStatus := domain.ConnPayFree
	if !isFree {
		payStatus = domain.ConnPayUnpaid
		pay := &domain.ConnPayment{
			UUID:    xuiCl.ID,
			UserID:  userID,
			AdminID: adminID,
			Status:  payStatus,
		}
		if err := uc.connPayRepo.Create(ctx, pay); err != nil {
			slog.Warn("connection: failed to save payment record", "uuid", xuiCl.ID, "err", err)
		}
	}

	return &domain.Connection{
		UUID:      xuiCl.ID,
		UserID:    userID,
		Label:     label,
		Link:      buildVLESSLink(xuiCl.ID, uc.xuiServerAddr, vlessBase, inbound.Port, label),
		IsActive:  true,
		PayStatus: payStatus,
		AdminID:   adminID,
	}, nil
}

func (uc *connectionUseCase) Remove(ctx context.Context, clientUUID string) error {
	if err := uc.xuiClient.DeleteClient(ctx, uc.xuiInboundID, clientUUID); err != nil {
		return fmt.Errorf("remove xui client: %w", err)
	}
	// Best-effort: remove payment record if exists.
	_ = uc.connPayRepo.SetStatus(ctx, clientUUID, domain.ConnPayFree)
	return nil
}

func (uc *connectionUseCase) SetEnabled(ctx context.Context, clientUUID string, enabled bool) error {
	if err := uc.xuiClient.SetClientEnabled(ctx, uc.xuiInboundID, clientUUID, enabled); err != nil {
		return fmt.Errorf("xui toggle: %w", err)
	}
	return nil
}

func (uc *connectionUseCase) GetAllUnpaidPayments(ctx context.Context) ([]*domain.ConnPayment, error) {
	return uc.connPayRepo.GetAllUnpaid(ctx)
}

func (uc *connectionUseCase) GetOverduePayments(ctx context.Context, olderThan time.Duration) ([]*domain.ConnPayment, error) {
	return uc.connPayRepo.GetOverdue(ctx, time.Now().Add(-olderThan))
}

func (uc *connectionUseCase) GetAdminPaymentInfo(ctx context.Context, connUUID string) (string, error) {
	pay, err := uc.connPayRepo.GetByUUID(ctx, connUUID)
	if err != nil {
		return "", fmt.Errorf("get payment record: %w", err)
	}
	info, err := uc.connPayRepo.GetAdminPaymentInfo(ctx, pay.AdminID)
	if err != nil {
		return "", fmt.Errorf("get admin payment info: %w", err)
	}
	if info == "" {
		return "", fmt.Errorf("admin has not set payment info yet")
	}
	return info, nil
}

func (uc *connectionUseCase) SetPaymentPending(ctx context.Context, connUUID string) error {
	return uc.connPayRepo.SetStatus(ctx, connUUID, domain.ConnPayPending)
}

func (uc *connectionUseCase) ConfirmConnPayment(ctx context.Context, connUUID string) (int64, error) {
	pay, err := uc.connPayRepo.GetByUUID(ctx, connUUID)
	if err != nil {
		return 0, fmt.Errorf("get payment record: %w", err)
	}
	if err := uc.connPayRepo.SetStatus(ctx, connUUID, domain.ConnPayPaid); err != nil {
		return 0, fmt.Errorf("set paid: %w", err)
	}
	return pay.UserID, nil
}

func (uc *connectionUseCase) SetAdminPaymentInfo(ctx context.Context, adminID int64, info string) error {
	return uc.connPayRepo.SetAdminPaymentInfo(ctx, adminID, info)
}

func (uc *connectionUseCase) GetAdminOwnPaymentInfo(ctx context.Context, adminID int64) (string, error) {
	return uc.connPayRepo.GetAdminPaymentInfo(ctx, adminID)
}

func (uc *connectionUseCase) SetConnLastPaidAt(ctx context.Context, connUUID string, userID, adminID int64, paidAt *time.Time) error {
	return uc.connPayRepo.SetLastPaidAt(ctx, connUUID, userID, adminID, paidAt)
}

func (uc *connectionUseCase) GetConnsWithDueReminder(ctx context.Context) ([]*domain.ConnPayment, error) {
	return uc.connPayRepo.GetConnsWithDuePaidReminder(ctx)
}

// inboundAndVLESSBase parses stream settings from an already-fetched inbound.
func (uc *connectionUseCase) inboundAndVLESSBase(inbound *xui.Inbound) (*xui.Inbound, url.Values, error) {
	var stream xui.StreamSettings
	if err := json.Unmarshal([]byte(inbound.StreamSettings), &stream); err != nil {
		return nil, nil, fmt.Errorf("parse streamSettings: %w", err)
	}
	if stream.RealitySettings == nil {
		return nil, nil, fmt.Errorf("inbound has no realitySettings")
	}
	reality := stream.RealitySettings
	if len(reality.ServerNames) == 0 || len(reality.ShortIds) == 0 {
		return nil, nil, fmt.Errorf("reality config is missing serverNames or shortIds")
	}

	q := url.Values{}
	q.Set("type", "tcp")
	q.Set("security", "reality")
	q.Set("pbk", reality.Settings.PublicKey)
	q.Set("fp", reality.Settings.Fingerprint)
	q.Set("sni", reality.ServerNames[0])
	q.Set("sid", reality.ShortIds[0])
	q.Set("flow", "xtls-rprx-vision")
	return inbound, q, nil
}

func buildVLESSLink(clientUUID, serverAddr string, q url.Values, port int, label string) string {
	return fmt.Sprintf("vless://%s@%s:%d?%s#%s",
		clientUUID,
		serverAddr,
		port,
		q.Encode(),
		url.PathEscape(label),
	)
}

func sanitizeEmail(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}

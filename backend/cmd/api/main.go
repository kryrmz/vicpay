// Command api is the VicPay backend HTTP server. It wires configuration, the
// database pools (a direct one for migrations, an app one for serving), the
// ledger engine, and the authentication and wallet handlers behind a chi router
// with security middleware mounted from the start.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"github.com/vicpay/backend/internal/auth"
	"github.com/vicpay/backend/internal/authstore"
	"github.com/vicpay/backend/internal/config"
	"github.com/vicpay/backend/internal/database"
	"github.com/vicpay/backend/internal/ledger"
	"github.com/vicpay/backend/internal/middleware"
	"github.com/vicpay/backend/internal/otp"
	"github.com/vicpay/backend/internal/pii"
	"github.com/vicpay/backend/internal/transfer"
	"github.com/vicpay/backend/internal/wallet"
	pkgjwt "github.com/vicpay/backend/pkg/jwt"
	"github.com/vicpay/backend/pkg/response"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if err := run(logger); err != nil {
		logger.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg := config.Load()
	if err := cfg.ValidateForProduction(); err != nil {
		return err
	}
	ctx := context.Background()

	// Migrations run against a DIRECT connection (session advisory lock), never
	// through a transactional pooler.
	directPool, err := database.NewPool(ctx, cfg.DatabaseDirectURL, 4)
	if err != nil {
		return err
	}
	defer directPool.Close()
	if err := database.Migrate(ctx, directPool, migrationsDir()); err != nil {
		return err
	}
	logger.Info("migrations applied")

	// In a single-host deploy the schema is applied (as the owner) before the
	// least-privilege app role exists; this mode exits right after migrating.
	if cfg.MigrateOnly {
		logger.Info("migrate-only mode: exiting after migrations")
		return nil
	}

	// Application traffic uses the app pool, which may go through PgBouncer.
	appPool, err := database.NewPool(ctx, cfg.DatabaseURL, 25)
	if err != nil {
		return err
	}
	defer appPool.Close()

	piiKey, err := cfg.PIIKey()
	if err != nil {
		return err
	}
	cipher, err := pii.NewCipher(piiKey)
	if err != nil {
		return err
	}

	jwtMgr := pkgjwt.NewManager(cfg.JWTSecret, cfg.JWTIssuer, cfg.AccessTTL)
	otpStore := authstore.NewOTPStore(appPool)
	otpSender := authstore.NewLogSender(cfg.OTPDevEcho && cfg.IsDevelopment(), logger)
	otpSvc := otp.NewService(otpStore, otpSender, otp.DefaultConfig)

	authRepo := auth.NewRepository(appPool)
	authSvc := auth.NewService(authRepo, cipher, jwtMgr, otpSvc, cfg.RefreshTTL, cfg.IdleTimeout)
	cookies := auth.NewCookieWriter(!cfg.IsDevelopment(), cfg.RefreshTTL)
	authHandler := auth.NewHandler(authSvc, cookies)

	engine := ledger.New(appPool)
	walletHandler := wallet.NewHandler(wallet.NewService(appPool))
	transferHandler := transfer.NewHandler(transfer.NewService(appPool, engine, cipher), cfg.IsDevelopment())

	router := buildRouter(cfg, logger, jwtMgr, authHandler, walletHandler, transferHandler)

	srv := &http.Server{
		Addr:              netAddr(cfg.Port),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return serve(ctx, srv, logger)
}

func buildRouter(
	cfg *config.Config,
	logger *slog.Logger,
	jwtMgr *pkgjwt.Manager,
	authHandler *auth.Handler,
	walletHandler *wallet.Handler,
	transferHandler *transfer.Handler,
) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestIDMiddleware)
	r.Use(middleware.Recoverer(logger))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Request-Id"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	authLimiter := middleware.NewRateLimiter(20, time.Minute)
	csrf := middleware.CSRF(cfg.CORSOrigins)

	// Public auth endpoints (rate limited).
	r.Group(func(pub chi.Router) {
		pub.Use(authLimiter.Middleware)
		pub.Post("/auth/register", authHandler.Register)
		pub.Post("/auth/verify-phone", authHandler.VerifyPhone)
		pub.Post("/auth/resend-code", authHandler.ResendCode)
		pub.Post("/auth/login", authHandler.Login)
	})

	// Cookie-authenticated mutations: CSRF-guarded from day one.
	r.Group(func(cookie chi.Router) {
		cookie.Use(authLimiter.Middleware)
		cookie.Use(csrf)
		cookie.Post("/auth/refresh", authHandler.Refresh)
		cookie.Post("/auth/logout", authHandler.Logout)
	})

	// Bearer-authenticated resources.
	r.Group(func(protected chi.Router) {
		protected.Use(middleware.Authenticator(jwtMgr))
		protected.Get("/me", authHandler.Me)
		protected.Get("/wallets", walletHandler.Balances)
		protected.Get("/transactions", walletHandler.Transactions)
		protected.Post("/transfers", transferHandler.Transfer)
		protected.Post("/wallets/topup", transferHandler.TopUp)
	})

	return r
}

func serve(ctx context.Context, srv *http.Server, logger *slog.Logger) error {
	shutdownCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-shutdownCtx.Done():
		logger.Info("shutting down")
		graceCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(graceCtx)
	}
}

func migrationsDir() string {
	if d := os.Getenv("MIGRATIONS_DIR"); d != "" {
		return d
	}
	return "./migrations"
}

func netAddr(port int) string {
	return ":" + strconv.Itoa(port)
}

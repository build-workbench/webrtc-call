package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	sig "github.com/build-workbench/webrtc-call/internal/signal"
)

func parseOrigins(raw string) (origins []string, allowAll bool) {
	if raw == "" {
		return nil, false
	}
	if raw == "*" {
		return nil, true
	}
	for _, p := range strings.Split(raw, ",") {
		if p = strings.TrimSpace(p); p != "" {
			origins = append(origins, p)
		}
	}
	return origins, false
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func main() {
	addr := ":8080"
	if v := os.Getenv("ADDR"); v != "" {
		addr = v
	}

	wsAllowed, wsAllowAll := parseOrigins(os.Getenv("WS_ALLOWED_ORIGINS"))
	// 与 webrtc-signaling 共享同一 JWT 密钥:配置后 WS 必须携带 signaling
	// 签发的 join-token 才能连接(房间由 token 绑定)。未配置则跳过校验(本地开发)。
	hub := sig.NewHubWithOptions(sig.Options{
		AllowedOrigins:  wsAllowed,
		AllowAllOrigins: wsAllowAll,
		AuthSecret:      []byte(os.Getenv("SIGNAL_JWT_SECRET")),
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.HandleWS)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if hub.IsClosed() {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(w, "unavailable\n")
			return
		}
		_, _ = io.WriteString(w, "ok\n")
	})
	mux.Handle("/", http.FileServer(http.Dir("web")))

	srv := &http.Server{
		Addr:              addr,
		Handler:           securityHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Println("server: listening", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-quit
	log.Println("server: shutting down ...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	hub.Close()
	if err := srv.Shutdown(ctx); err != nil {
		log.Print("server: forced shutdown:", err)
		cancel()
		os.Exit(1)
	}
	cancel()
	log.Println("server: stopped")
}

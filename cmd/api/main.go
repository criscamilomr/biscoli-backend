package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/xxkmi/biscoli-backend/internal/database"
	"github.com/xxkmi/biscoli-backend/internal/handlers"
	"github.com/xxkmi/biscoli-backend/internal/notifications"
	"github.com/xxkmi/biscoli-backend/internal/wompi"
)

func main() {
	// Intentamos cargar el archivo .env si existe
	if err := godotenv.Load(); err != nil {
		log.Println("Nota: No se encontró archivo .env o no se pudo cargar, leyendo variables del entorno del sistema.")
	}

	// Initialize secrets from environment variables
	// In a real application, you should handle errors if these are missing
	integritySecret := os.Getenv("WOMPI_INTEGRITY_SECRET")
	eventsSecret := os.Getenv("WOMPI_EVENTS_SECRET")
	privateKey := os.Getenv("WOMPI_PRIVATE_KEY")
	publicKey := os.Getenv("WOMPI_PUBLIC_KEY")
	environment := os.Getenv("WOMPI_ENVIRONMENT")
	databaseURL := os.Getenv("DATABASE_URL")

	if integritySecret == "" {
		log.Println("WARNING: WOMPI_INTEGRITY_SECRET is not set")
		integritySecret = "test_integrity_secret" // fallback for local testing
	}

	if databaseURL == "" {
		log.Println("WARNING: DATABASE_URL is not set. Database operations will fail.")
	}

	ctx := context.Background()

	// 0. Inicializar Base de Datos
	db, err := database.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("Error connecting to database: %v\n", err)
	}
	defer db.Close()
	log.Println("Conectado a PostgreSQL exitosamente")

	// 1. Inicializar servicios
	wompiService := wompi.NewService(integritySecret, eventsSecret, privateKey, publicKey, environment)

	// 2. Inicializar servicio de notificaciones
	var notifiers []notifications.Notifier

	// Email notifier (optional - only if SMTP credentials are configured)
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")
	notificationEmail := os.Getenv("NOTIFICATION_EMAIL")

	if smtpUser != "" && smtpPass != "" {
		emailNotifier := notifications.NewEmailNotifier(smtpHost, smtpPort, smtpUser, smtpPass, notificationEmail)
		notifiers = append(notifiers, emailNotifier)
		log.Printf("📧 Notificaciones por email activadas → %s\n", notificationEmail)
	} else {
		log.Println("⚠️ Notificaciones por email desactivadas (SMTP_USER/SMTP_PASS no configurados)")
	}

	// Telegram notifier (optional - only if bot credentials are configured)
	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	telegramChatID := os.Getenv("TELEGRAM_CHAT_ID")

	if telegramToken != "" && telegramChatID != "" {
		telegramNotifier := notifications.NewTelegramNotifier(telegramToken, telegramChatID)
		notifiers = append(notifiers, telegramNotifier)
		log.Println("📱 Notificaciones por Telegram activadas")
	} else {
		log.Println("⚠️ Notificaciones por Telegram desactivadas (TELEGRAM_BOT_TOKEN/TELEGRAM_CHAT_ID no configurados)")
	}

	notificationService := notifications.NewService(notifiers...)

	// 3. Inicializar handlers
	paymentHandler := handlers.NewPaymentHandler(wompiService, db, notificationService)

	// 3. Configurar rutas HTTP
	mux := http.NewServeMux()
	
	// Endpoint invocado desde Angular para preparar el pago (Widget)
	mux.HandleFunc("/api/checkout", paymentHandler.HandleCheckout)
	
	// Endpoint para crear un pago directamente mediante API
	mux.HandleFunc("/api/payments", paymentHandler.HandleCreatePayment)

	// Endpoint para consultar el estado del pago mediante API
	// Usamos /api/payments/ como prefijo, el handler extraerá el ID
	mux.HandleFunc("/api/payments/", paymentHandler.HandleGetPaymentStatus)

	// Endpoint público (Webhook) para que Wompi notifique el estado real del pago
	mux.HandleFunc("/api/webhooks/wompi", paymentHandler.HandleWompiWebhook)

	// Opcional: Endpoint simple para verificar que el servidor está corriendo
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 4. Iniciar el servidor
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	serverAddr := fmt.Sprintf(":%s", port)
	log.Printf("Iniciando servidor de Biscoli en el puerto %s...\n", port)
	
	if err := http.ListenAndServe(serverAddr, mux); err != nil {
		log.Fatalf("Error al iniciar el servidor: %v", err)
	}
}

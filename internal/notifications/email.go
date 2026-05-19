package notifications

import (
	"fmt"
	"net/smtp"
	"strings"
)

// EmailNotifier sends order notifications via SMTP email.
type EmailNotifier struct {
	Host     string // SMTP host (e.g. "smtp.gmail.com")
	Port     string // SMTP port (e.g. "587")
	User     string // SMTP username (email address)
	Password string // SMTP password or App Password
	To       string // Recipient email for notifications
}

// NewEmailNotifier creates a new EmailNotifier with the given SMTP configuration.
func NewEmailNotifier(host, port, user, password, to string) *EmailNotifier {
	return &EmailNotifier{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		To:       to,
	}
}

// Name returns the notifier channel name.
func (e *EmailNotifier) Name() string {
	return "Email"
}

// Send sends an order notification email.
func (e *EmailNotifier) Send(order OrderNotification) error {
	total := fmt.Sprintf("$%s %s", formatMoney(order.TotalAmount), order.Currency)
	subject := fmt.Sprintf("🍪 Nuevo Pedido Biscoli - %s - %s", order.CustomerName, total)

	body := buildHTMLEmail(order)

	msg := "From: Biscoli Pedidos <" + e.User + ">\r\n" +
		"To: " + e.To + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=\"UTF-8\"\r\n" +
		"\r\n" +
		body

	auth := smtp.PlainAuth("", e.User, e.Password, e.Host)
	addr := e.Host + ":" + e.Port

	return smtp.SendMail(addr, auth, e.User, []string{e.To}, []byte(msg))
}

// buildHTMLEmail creates a nicely formatted HTML email for the order.
func buildHTMLEmail(order OrderNotification) string {
	total := fmt.Sprintf("$%s %s", formatMoney(order.TotalAmount), order.Currency)

	// Build items list
	var itemsHTML strings.Builder
	for _, item := range order.Items {
		itemsHTML.WriteString(fmt.Sprintf(
			`<tr><td style="padding:8px 12px;border-bottom:1px solid #eee;font-size:14px;">%s</td>
			 <td style="padding:8px 12px;border-bottom:1px solid #eee;font-size:14px;text-align:center;">x%d</td></tr>`,
			item.Name, item.Quantity,
		))
	}

	notesSection := ""
	if order.Notes != "" {
		notesSection = fmt.Sprintf(`
			<tr><td style="padding:6px 0;color:#666;font-size:13px;">📝 Notas:</td>
			<td style="padding:6px 0;font-size:13px;font-style:italic;">%s</td></tr>`, order.Notes)
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="margin:0;padding:0;background-color:#f5f0eb;font-family:Arial,Helvetica,sans-serif;">
<div style="max-width:600px;margin:20px auto;background:white;border-radius:16px;overflow:hidden;box-shadow:0 4px 20px rgba(0,0,0,0.1);">
  
  <!-- Header -->
  <div style="background-color:#5C2E35;padding:24px;text-align:center;">
    <h1 style="color:white;margin:0;font-size:24px;">🍪 Nuevo Pedido Biscoli</h1>
    <p style="color:#F5EDE2;margin:8px 0 0;font-size:14px;">Referencia: %s</p>
  </div>

  <!-- Total Banner -->
  <div style="background-color:#4A5D4A;padding:16px;text-align:center;">
    <span style="color:#e0eadc;font-size:14px;font-weight:bold;">TOTAL PAGADO</span>
    <div style="color:white;font-size:28px;font-weight:900;margin-top:4px;">%s</div>
    <span style="color:#e0eadc;font-size:12px;">Método: %s</span>
  </div>

  <div style="padding:24px;">
    <!-- Customer Info -->
    <h2 style="color:#5C2E35;font-size:16px;margin:0 0 12px;border-bottom:2px solid #F5EDE2;padding-bottom:8px;">👤 Datos del Cliente</h2>
    <table style="width:100%%;margin-bottom:20px;">
      <tr><td style="padding:6px 0;color:#666;font-size:13px;width:100px;">Nombre:</td><td style="padding:6px 0;font-size:13px;font-weight:bold;">%s</td></tr>
      <tr><td style="padding:6px 0;color:#666;font-size:13px;">Email:</td><td style="padding:6px 0;font-size:13px;">%s</td></tr>
      <tr><td style="padding:6px 0;color:#666;font-size:13px;">Teléfono:</td><td style="padding:6px 0;font-size:13px;font-weight:bold;">%s</td></tr>
    </table>

    <!-- Delivery Info -->
    <h2 style="color:#5C2E35;font-size:16px;margin:0 0 12px;border-bottom:2px solid #F5EDE2;padding-bottom:8px;">📦 Datos de Entrega</h2>
    <table style="width:100%%;margin-bottom:20px;">
      <tr><td style="padding:6px 0;color:#666;font-size:13px;width:100px;">Ciudad:</td><td style="padding:6px 0;font-size:13px;">%s</td></tr>
      <tr><td style="padding:6px 0;color:#666;font-size:13px;">Dirección:</td><td style="padding:6px 0;font-size:13px;font-weight:bold;">%s</td></tr>
      <tr><td style="padding:6px 0;color:#666;font-size:13px;">Barrio:</td><td style="padding:6px 0;font-size:13px;">%s</td></tr>
      %s
    </table>

    <!-- Products -->
    <h2 style="color:#5C2E35;font-size:16px;margin:0 0 12px;border-bottom:2px solid #F5EDE2;padding-bottom:8px;">🛒 Productos</h2>
    <table style="width:100%%;border-collapse:collapse;">
      <thead>
        <tr style="background-color:#f5f0eb;">
          <th style="padding:8px 12px;text-align:left;font-size:12px;color:#666;text-transform:uppercase;">Producto</th>
          <th style="padding:8px 12px;text-align:center;font-size:12px;color:#666;text-transform:uppercase;">Cantidad</th>
        </tr>
      </thead>
      <tbody>
        %s
      </tbody>
    </table>
  </div>

  <!-- Footer -->
  <div style="background-color:#f5f0eb;padding:16px;text-align:center;">
    <p style="color:#999;font-size:11px;margin:0;">Este email fue generado automáticamente por el sistema de Biscoli.</p>
  </div>
</div>
</body>
</html>`,
		order.Reference,
		total,
		order.PaymentMethod,
		order.CustomerName,
		order.Email,
		order.Phone,
		order.City,
		order.Address,
		order.Neighborhood,
		notesSection,
		itemsHTML.String(),
	)
}

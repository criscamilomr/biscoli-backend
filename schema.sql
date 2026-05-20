-- schema.sql
-- Este archivo contiene las definiciones de las tablas para la integración con Wompi usando PostgreSQL.

-- Tabla para almacenar los pedidos (orders/transactions)
CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    reference VARCHAR(255) UNIQUE NOT NULL,    -- Ej. BISCOLI-1234
    wompi_transaction_id VARCHAR(255),         -- El ID real que asigna Wompi (ej. 01-123...)
    amount_in_cents BIGINT NOT NULL,           -- Monto en centavos
    currency VARCHAR(10) NOT NULL DEFAULT 'COP',
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING', -- PENDING, APPROVED, DECLINED, ERROR
    customer_email VARCHAR(255),               -- Email del cliente
    order_details JSONB,                       -- Detalles del pedido (nombre, dirección, items)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tabla para almacenar los webhooks crudos de Wompi, útil para auditoría
CREATE TABLE IF NOT EXISTS wompi_webhooks (
    id SERIAL PRIMARY KEY,
    transaction_id VARCHAR(255),               -- Extraído del webhook si es posible
    event_type VARCHAR(255),                   -- Ej. transaction.updated
    payload JSONB NOT NULL,                    -- Todo el contenido del webhook
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Índices para búsquedas rápidas
CREATE INDEX IF NOT EXISTS idx_orders_reference ON orders(reference);
CREATE INDEX IF NOT EXISTS idx_orders_wompi_transaction_id ON orders(wompi_transaction_id);
CREATE INDEX IF NOT EXISTS idx_wompi_webhooks_transaction_id ON wompi_webhooks(transaction_id);

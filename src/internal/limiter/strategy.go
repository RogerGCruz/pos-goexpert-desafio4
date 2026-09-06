package limiter

import (
	"context"
	"time"
)

// Strategy define o contrato de persistência para o Rate Limiter.
// Implementações alternativas (memória, banco relacional, etc.) podem
// substituir o Redis sem alterar a lógica de negócio em Limiter.
type Strategy interface {
	// Increment incrementa o contador de requisições da chave dentro da
	// janela informada e retorna o total acumulado após o incremento.
	Increment(ctx context.Context, key string, window time.Duration) (int64, error)

	// IsBlocked informa se a chave está atualmente em período de bloqueio.
	IsBlocked(ctx context.Context, key string) (bool, error)

	// Block bloqueia a chave pelo período informado.
	Block(ctx context.Context, key string, duration time.Duration) error
}

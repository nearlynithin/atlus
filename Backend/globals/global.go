package globals

import(
	"github.com/jackc/pgx/v5/pgxpool"
)

var Sessions = map[string]string{}

var DB *pgxpool.Pool

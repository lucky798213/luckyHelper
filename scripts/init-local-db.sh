#!/usr/bin/env bash
#
# 在本机已存在的 PostgreSQL 上，创建 luckyHelper 所需的角色与数据库。
#
# 适用场景：本机 5432 端口已被另一个 PostgreSQL 占用（例如 EDB 官方安装包装的
# PostgreSQL 18，装在 /Library/PostgreSQL/18），此时 docker compose 里的 postgres
# 起不来（端口冲突），又不想折腾 Docker，就直接复用这个实例。
#
# 只需要建「角色 + 库」；业务表由后端启动时的 postgres.BootstrapSchema 幂等创建。
#
# 用法：
#   ./scripts/init-local-db.sh                    # 自动探测 psql
#   PSQL=/path/to/psql ./scripts/init-local-db.sh # 手动指定 psql
#   SUPERUSER=postgres ./scripts/init-local-db.sh
#
set -euo pipefail

DB_NAME="${DB_NAME:-aiagent}"
DB_USER="${DB_USER:-aiagent}"
DB_PASSWORD="${DB_PASSWORD:-aiagent123}"
PG_HOST="${PG_HOST:-localhost}"
PG_PORT="${PG_PORT:-5432}"
SUPERUSER="${SUPERUSER:-postgres}"

# ── 探测 psql ────────────────────────────────────────────────
if [ -z "${PSQL:-}" ]; then
  for cand in /Library/PostgreSQL/*/bin/psql /opt/homebrew/opt/postgresql*/bin/psql /usr/local/opt/postgresql*/bin/psql; do
    if [ -x "$cand" ]; then PSQL="$cand"; break; fi
  done
fi
if [ -z "${PSQL:-}" ] || [ ! -x "$PSQL" ]; then
  echo "❌ 找不到 psql。请用 PSQL=/path/to/psql 指定，例如：" >&2
  echo "   PSQL=/Library/PostgreSQL/18/bin/psql $0" >&2
  exit 1
fi

echo "==> 使用 psql: $PSQL"
echo "==> 连接 $PG_HOST:$PG_PORT 以超级用户 $SUPERUSER 创建 role=$DB_USER db=$DB_NAME"
echo "    （下一步会提示输入 $SUPERUSER 的密码）"

# ── 1) 建角色（幂等）─────────────────────────────────────────
# 注意：CREATE DATABASE 不能在事务块内执行，因此每条语句必须用独立的 -c 下发，
# 不能把两条 SQL 用分号拼在一个 -c 里（那样会被当成一个隐式事务而报错）。
"$PSQL" -h "$PG_HOST" -p "$PG_PORT" -U "$SUPERUSER" -d postgres -v ON_ERROR_STOP=1 \
  -c "DO \$\$
      BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '$DB_USER') THEN
          CREATE ROLE $DB_USER LOGIN PASSWORD '$DB_PASSWORD';
          RAISE NOTICE 'created role %', '$DB_USER';
        ELSE
          ALTER ROLE $DB_USER LOGIN PASSWORD '$DB_PASSWORD';
          RAISE NOTICE 'role % already existed, password reset', '$DB_USER';
        END IF;
      END
      \$\$;"

# ── 2) 建库（幂等）───────────────────────────────────────────
exists=$("$PSQL" -h "$PG_HOST" -p "$PG_PORT" -U "$SUPERUSER" -d postgres -tAc \
  "SELECT 1 FROM pg_database WHERE datname = '$DB_NAME'")
if [ "$exists" = "1" ]; then
  echo "==> 数据库 $DB_NAME 已存在，跳过创建"
else
  "$PSQL" -h "$PG_HOST" -p "$PG_PORT" -U "$SUPERUSER" -d postgres -v ON_ERROR_STOP=1 \
    -c "CREATE DATABASE $DB_NAME OWNER $DB_USER;"
  echo "==> 已创建数据库 $DB_NAME（owner: $DB_USER）"
fi

# ── 3) 验证以业务账号登录 ────────────────────────────────────
echo "==> 验证 $DB_USER@$DB_NAME 可登录"
PGPASSWORD="$DB_PASSWORD" "$PSQL" -h "$PG_HOST" -p "$PG_PORT" -U "$DB_USER" -d "$DB_NAME" \
  -c "SELECT current_user AS \"user\", current_database() AS db;"

echo
echo "✅ 完成。现在重启后端（go run ./cmd/server），"
echo "   启动日志里 PostgreSQL 应显示 connected，并自动创建全部业务表。"

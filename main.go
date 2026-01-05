package main

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"log"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

type Config struct {
	OldDSN string
	NewDSN string
}

var config Config

func main() {
	if len(os.Args) > 2 {

		config.OldDSN = os.Args[1]
		config.NewDSN = os.Args[2]
	} else {
		fmt.Println("⚠️命令参数中未查询到数据库连接信息，将从环境变量获取⚠️")
		fmt.Println("⚠️环境变量ONEAPI_SOURCE_SQL_DSN:MartialBE/one-hub数据库的连接字符串(源)⚠️")
		fmt.Println("⚠️环境变量ONEAPI_TARGET_SQL_DSN:songquanpeng/one-api数据库的连接字符串(目标)⚠️")
		config = loadConfig()
	}

	oldDB := openDatabase(config.OldDSN)
	newDB := openDatabase(config.NewDSN)

	tables := []string{"channels", "logs", "options", "redemptions", "tokens", "users", "abilities"}
	fmt.Println("🚩数据处理开始🚩")
	fmt.Println("======================")
	for _, table := range tables {
		fmt.Printf("🚀 正在处理表: %s\n", table)
		migrateTable(oldDB, newDB, table)
		fmt.Printf("✅ 完成处理表: %s\n", table)
	}

	if boolEnvDefaultTrue("ONEAPI_REBUILD_ABILITIES") {
		fmt.Println("======================")
		fmt.Println("🔧 正在尝试重建目标库 abilities（从目标库 channels 派生）")
		rebuildTargetAbilitiesFromChannels(newDB)
	}
	fmt.Println("======================")
	fmt.Println("🚩数据处理完成🚩")
}

func boolEnvDefaultTrue(name string) bool {
	val, ok := os.LookupEnv(name)
	if !ok {
		return true
	}
	val = strings.TrimSpace(strings.ToLower(val))
	if val == "" {
		return true
	}
	switch val {
	case "0", "false", "no", "off" :
		return false
	default:
		return true
	}
}

func rebuildTargetAbilitiesFromChannels(newDB *sql.DB) {
	newDriver, _ := detectDriver(config.NewDSN)
	abilityCols := getColumns(newDB, "abilities", newDriver)
	if len(abilityCols) == 0 {
		fmt.Println("⚠️ 目标库中没有找到表: abilities，跳过重建")
		return
	}

	channelCols := getColumns(newDB, "channels", newDriver)
	if len(channelCols) == 0 {
		fmt.Println("⚠️ 目标库中没有找到表: channels，无法重建 abilities")
		return
	}

	required := []string{"id", "group", "models", "status"}
	for _, col := range required {
		if !contains(channelCols, col) {
			fmt.Printf("⚠️ 目标库 channels 缺少字段 %s，跳过重建 abilities\n", col)
			return
		}
	}

	priorityExpr := "0"
	if contains(channelCols, "priority") {
		priorityExpr = quoteIdent(newDriver, "priority")
	}

	query := fmt.Sprintf(
		"SELECT %s,%s,%s,%s,%s FROM %s",
		quoteIdent(newDriver, "id"),
		quoteIdent(newDriver, "group"),
		quoteIdent(newDriver, "models"),
		quoteIdent(newDriver, "status"),
		priorityExpr,
		quoteIdent(newDriver, "channels"),
	)

	rows, err := newDB.Query(query)
	if err != nil {
		fmt.Printf("⚠️ 查询目标库 channels 失败，无法重建 abilities: %v\n", err)
		return
	}
	defer rows.Close()

	insertColumns := []string{"group", "model", "channel_id", "enabled", "priority"}
	const maxBatchRows = 500

	tx, err := newDB.Begin()
	if err != nil {
		fmt.Printf("⚠️ 开启事务失败（重建 abilities）: %v\n", err)
		return
	}

	flush := func(batchArgs []any, batchRows int) error {
		if batchRows == 0 {
			return nil
		}
		insertSQL := buildBulkInsertSQL("abilities", insertColumns, newDriver, batchRows)
		_, err := tx.Exec(insertSQL, batchArgs...)
		return err
	}

	var (
		batchArgs []any
		batchRows int
		seenRows  int
		inserted  int
	)

	for rows.Next() {
		var (
			channelID int
			group     sql.NullString
			models    sql.NullString
			status    sql.NullInt64
			priority  sql.NullInt64
		)
		err := rows.Scan(&channelID, &group, &models, &status, &priority)
		if err != nil {
			_ = tx.Rollback()
			fmt.Printf("⚠️ 扫描目标库 channels 失败，重建 abilities 中止: %v\n", err)
			return
		}
		seenRows++

		groups := splitCSVTrim(group.String)
		modelsList := splitCSVTrim(models.String)
		if len(groups) == 0 || len(modelsList) == 0 {
			continue
		}
		groups = dedupStrings(groups)
		modelsList = dedupStrings(modelsList)

		enabled := status.Valid && status.Int64 == 1
		var prio any
		if priority.Valid {
			prio = priority.Int64
		} else {
			prio = nil
		}

		for _, g := range groups {
			for _, m := range modelsList {
				batchArgs = append(batchArgs, g, m, channelID, enabled, prio)
				batchRows++
				inserted++
				if batchRows >= maxBatchRows {
					err := flush(batchArgs, batchRows)
					if err != nil {
						_ = tx.Rollback()
						fmt.Printf("⚠️ 重建 abilities 批量写入失败: %v\n", err)
						return
					}
					batchArgs = batchArgs[:0]
					batchRows = 0
				}
			}
		}
		if seenRows%200 == 0 {
			fmt.Printf("⏳ 已扫描 channels %d 行\n", seenRows)
		}
	}

	if err := flush(batchArgs, batchRows); err != nil {
		_ = tx.Rollback()
		fmt.Printf("⚠️ 重建 abilities 批量写入失败: %v\n", err)
		return
	}

	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		fmt.Printf("⚠️ 提交事务失败（重建 abilities）: %v\n", err)
		return
	}

	fmt.Printf("✅ abilities 重建完成：扫描 channels=%d，生成写入行=%d（重复键会被忽略）\n", seenRows, inserted)
}

func splitCSVTrim(s string) []string {
	parts := strings.Split(s, ",")
	res := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		res = append(res, p)
	}
	return res
}

func dedupStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	res := make([]string, 0, len(items))
	for _, it := range items {
		if _, ok := seen[it]; ok {
			continue
		}
		seen[it] = struct{}{}
		res = append(res, it)
	}
	return res
}

func buildBulkInsertSQL(table string, columns []string, driver string, rows int) string {
	if rows <= 0 {
		return ""
	}
	quotedCols := make([]string, 0, len(columns))
	for _, col := range columns {
		quotedCols = append(quotedCols, quoteIdent(driver, col))
	}

	valuesPlaceholder := buildValuesPlaceholders(driver, len(columns), rows)
	tableIdent := quoteIdent(driver, table)

	switch driver {
	case "mysql":
		return fmt.Sprintf("INSERT IGNORE INTO %s (%s) VALUES %s", tableIdent, strings.Join(quotedCols, ","), valuesPlaceholder)
	case "sqlite":
		return fmt.Sprintf("INSERT OR IGNORE INTO %s (%s) VALUES %s", tableIdent, strings.Join(quotedCols, ","), valuesPlaceholder)
	case "postgres":
		return fmt.Sprintf("INSERT INTO %s (%s) VALUES %s ON CONFLICT DO NOTHING", tableIdent, strings.Join(quotedCols, ","), valuesPlaceholder)
	default:
		log.Fatalf("不支持的数据库驱动: %s", driver)
		return ""
	}
}

func buildValuesPlaceholders(driver string, cols int, rows int) string {
	if cols <= 0 || rows <= 0 {
		return ""
	}
	if driver != "postgres" {
		row := "(" + strings.TrimSuffix(strings.Repeat("?,", cols), ",") + ")"
		return strings.TrimSuffix(strings.Repeat(row+",", rows), ",")
	}
	// postgres: ($1,$2,...),($n+1,...)
	parts := make([]string, 0, rows)
	arg := 1
	for i := 0; i < rows; i++ {
		rowParts := make([]string, 0, cols)
		for j := 0; j < cols; j++ {
			rowParts = append(rowParts, "$"+strconv.Itoa(arg))
			arg++
		}
		parts = append(parts, "("+strings.Join(rowParts, ",")+")")
	}
	return strings.Join(parts, ",")
}

func loadConfig() Config {
	return Config{
		OldDSN: os.Getenv("ONEAPI_SOURCE_SQL_DSN"),
		NewDSN: os.Getenv("ONEAPI_TARGET_SQL_DSN"),
	}
}

func openDatabase(dsn string) *sql.DB {
	driver, dsn := detectDriver(dsn)
	db, err := sql.Open(driver, dsn)
	if err != nil {
		log.Fatalf("无法连接到数据库: %v", err)
	}
	return db
}

func detectDriver(dsn string) (string, string) {
	dsn = strings.TrimSpace(dsn)

	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		// lib/pq 支持 URL 形式 DSN，不能把 scheme 截掉
		return "postgres", dsn
	}
	if strings.HasPrefix(dsn, "mysql://") {
		normalized, err := normalizeMySQLURL(dsn)
		if err == nil {
			return "mysql", normalized
		}
		// 回退：至少别直接把 scheme 截断导致更诡异的错误
		return "mysql", strings.TrimPrefix(dsn, "mysql://")
	}

	// 兼容：不带 scheme 的 DSN
	if looksLikePostgresConnString(dsn) {
		return "postgres", dsn
	}
	if looksLikeMySQLDSN(dsn) {
		return "mysql", dsn
	}

	return "sqlite", dsn
}

func looksLikePostgresConnString(dsn string) bool {
	// 典型 pq conn string: "host=... user=... password=... dbname=... sslmode=..."
	return strings.Contains(dsn, "host=") || strings.Contains(dsn, "sslmode=")
}

func looksLikeMySQLDSN(dsn string) bool {
	// 典型 go-sql-driver/mysql DSN: user:pass@tcp(host:3306)/db?parseTime=true
	return strings.Contains(dsn, "@tcp(") || (strings.Contains(dsn, "@") && strings.Contains(dsn, ")/") ) || (strings.Contains(dsn, "@") && strings.Contains(dsn, "/"))
}

func normalizeMySQLURL(dsn string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	user := ""
	pass := ""
	if u.User != nil {
		user = u.User.Username()
		pass, _ = u.User.Password()
	}
	dbName := strings.TrimPrefix(u.Path, "/")
	if dbName == "" {
		return "", fmt.Errorf("mysql url missing database name")
	}
	host := u.Host
	if host == "" {
		return "", fmt.Errorf("mysql url missing host")
	}
	if !strings.Contains(host, ":") {
		host = host + ":3306"
	}
	auth := user
	if user != "" && pass != "" {
		auth = user + ":" + pass
	}
	// user 允许为空（例如使用 socket / 免密场景），这里尽量宽松
	dsnCore := fmt.Sprintf("%s@tcp(%s)/%s", auth, host, dbName)
	dsnCore = strings.TrimPrefix(dsnCore, "@") // auth 为空时去掉前导 @
	if u.RawQuery != "" {
		dsnCore += "?" + u.RawQuery
	}
	return dsnCore, nil
}

func migrateTable(oldDB, newDB *sql.DB, table string) {
	oldDriver, _ := detectDriver(config.OldDSN)
	newDriver, _ := detectDriver(config.NewDSN)

	oldColumns := getColumns(oldDB, table, oldDriver)
	newColumns := getColumns(newDB, table, newDriver)

	if len(oldColumns) == 0 {
		fmt.Printf("⚠️ 源库中没有找到表: %s\n", table)
		return
	}

	if len(newColumns) == 0 {
		fmt.Printf("⚠️ 新库中没有找到表: %s\n", table)
		return
	}

	commonColumns := intersectPreserveOrder(newColumns, oldColumns)
	if len(commonColumns) == 0 {
		fmt.Printf("⚠️ 表 %s 没有可迁移的同名字段(源/目标列交集为空)，已跳过\n", table)
		return
	}

	missingColumns := findMissingColumns(oldColumns, newColumns)
	if len(missingColumns) > 0 {
		fmt.Printf("⚠️ 旧库中的表 %s 存在新库中没有的字段: %v\n", table, missingColumns)
	}

	rows, err := oldDB.Query(fmt.Sprintf("SELECT * FROM %s", quoteIdent(oldDriver, table)))
	if err != nil {
		fmt.Printf("⚠️ 查询源库表 %s 失败: %v\n", table, err)
		return
	}
	defer rows.Close()

	columns, _ := rows.Columns()
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}
	insertSQL := buildInsertSQL(table, commonColumns, newDriver)

	tx, err := newDB.Begin()
	if err != nil {
		fmt.Printf("⚠️ 开启事务失败: %v\n", err)
		return
	}

	count := 0
	for rows.Next() {
		err := rows.Scan(valuePtrs...)
		if err != nil {
			_ = tx.Rollback()
			fmt.Printf("⚠️ 扫描行数据失败: %v\n", err)
			return
		}
		insertValues := buildInsertValues(values, oldColumns, commonColumns, table)
		_, err = tx.Exec(insertSQL, insertValues...)
		if err != nil {
			_ = tx.Rollback()
			fmt.Printf("⚠️ 插入新库表 %s 失败: %v\n", table, err)
			return
		}
		count++
		if count%100 == 0 {
			fmt.Printf("⏳ 已处理 %d 行数据\n", count)
		}
	}

	err = tx.Commit()
	if err != nil {
		_ = tx.Rollback()
		fmt.Printf("⚠️ 提交事务失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 表 %s 迁移完成，共处理 %d 行数据\n", table, count)
}

func getColumns(db *sql.DB, table string, driver string) []string {
	// 用 LIMIT 0 取列名，避免实际读取数据
	rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s LIMIT 0", quoteIdent(driver, table)))
	if err != nil {
		return nil
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil
	}

	return columns
}

func findMissingColumns(oldColumns, newColumns []string) []string {
	missingColumns := []string{}
	for _, col := range oldColumns {
		if !contains(newColumns, col) {
			missingColumns = append(missingColumns, col)
		}
	}
	return missingColumns
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func buildInsertSQL(table string, columns []string, driver string) string {
	quotedCols := make([]string, 0, len(columns))
	for _, col := range columns {
		quotedCols = append(quotedCols, quoteIdent(driver, col))
	}

	placeholders := buildPlaceholders(driver, len(columns))
	tableIdent := quoteIdent(driver, table)

	switch driver {
	case "mysql":
		return fmt.Sprintf("INSERT IGNORE INTO %s (%s) VALUES (%s)", tableIdent, strings.Join(quotedCols, ","), placeholders)
	case "sqlite":
		return fmt.Sprintf("INSERT OR IGNORE INTO %s (%s) VALUES (%s)", tableIdent, strings.Join(quotedCols, ","), placeholders)
	case "postgres":
		return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON CONFLICT DO NOTHING", tableIdent, strings.Join(quotedCols, ","), placeholders)
	default:
		log.Fatalf("不支持的数据库驱动: %s", driver)
		return ""
	}
}

func buildPlaceholders(driver string, n int) string {
	if n <= 0 {
		return ""
	}
	if driver == "postgres" {
		parts := make([]string, 0, n)
		for i := 1; i <= n; i++ {
			parts = append(parts, "$"+strconv.Itoa(i))
		}
		return strings.Join(parts, ",")
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

func quoteIdent(driver, ident string) string {
	// ident 只来自固定表名/列名，不做复杂转义；如需支持特殊字符再扩展
	switch driver {
	case "postgres":
		return "\"" + ident + "\""
	default:
		return "`" + ident + "`"
	}
}

func buildInsertValues(values []interface{}, oldColumns, commonColumns []string, table string) []interface{} {
	insertValues := make([]interface{}, 0, len(commonColumns))
	for _, col := range commonColumns {
		idx := indexOf(oldColumns, col)
		if idx == -1 {
			// 理论上不会发生（commonColumns 是交集），但为健壮性保底
			insertValues = append(insertValues, getDefaultForType(reflect.TypeOf(values[0])))
			continue
		}
		value := values[idx]
		if table == "channels" && col == "type" {
			fmt.Println("🔗 处理渠道类别数据")
			value = upgradeChannelType(value)
		}
		insertValues = append(insertValues, value)
	}
	return insertValues
}

func intersectPreserveOrder(primary, secondary []string) []string {
	res := make([]string, 0, len(primary))
	for _, c := range primary {
		if contains(secondary, c) {
			res = append(res, c)
		}
	}
	return res
}

func indexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}

func getDefaultForType(t reflect.Type) interface{} {
	switch t.Kind() {
	case reflect.String:
		return ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return 0
	case reflect.Float32, reflect.Float64:
		return 0.0
	case reflect.Bool:
		return false
	case reflect.Slice, reflect.Map, reflect.Ptr, reflect.Interface:
		return nil
	default:
		return ""
	}
}
func BytesToInt(b []uint8) int {
	if len(b) < 4 {
		return 0
	}
	return int(binary.BigEndian.Uint32(b))
}

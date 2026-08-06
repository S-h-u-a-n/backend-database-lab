# 後端資料庫實作練習

本實作練習是陽明交通大學軟體開發社（NYCU Software Development Club，SDC）後端培訓課程的一部分，著重於選課系統的資料庫層。

完成本實作後，你將學會設計正規化的關聯式資料庫綱要（schema），使用 `schema.sql` 描述其目前結構，並透過有版本編號的 migration 逐步演進資料庫。

你也會使用 sqlc 實作型別安全的 CRUD 與 JOIN 查詢，接著透過 PostgreSQL 整合測試驗證查詢結果及資料庫約束。

> **課程講義：** 開始本實作前，請先閱讀 [DB CRUD](https://app.notion.com/p/sdc-nycu/DB-CRUD-3a97dadd804080aebf25cf742474470c?source=copy_link)。

> **需要協助嗎？** 請先自行嘗試完成練習。若遇到困難，可參考 [`reference-answer` 分支](https://github.com/ilsao/backend-database-lab/tree/reference-answer)，比較你的做法與可正常運作的實作。請繼續在你自己 fork 的分支上完成作業，不要提交參考答案分支。

## 如何開始

### Fork 並 Clone Repository

1. 開啟[原始 repository](https://github.com/ilsao/backend-database-lab)。
2. 點選右上角的 `Fork`，再點選 `Create fork`，將 repository 複製到你的 GitHub 帳號。
3. Clone 你的 fork 並進入專案目錄。請將 `<your-github-username>` 替換成你的 GitHub 使用者名稱：

   ```bash
   git clone https://github.com/<your-github-username>/backend-database-lab.git
   cd backend-database-lab
   ```

請確認你 clone 的是自己的 fork，而非原始 repository。如此才能將成果 push 到你自己的 GitHub repository。

如果你喜歡這份實作練習，也可以回到[原始 repository](https://github.com/ilsao/backend-database-lab) 並點選 `Star`。

### 設定 Docker

若你使用 Linux 或 WSL，請執行下列指令安裝 Docker：

```bash
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER
```

若你使用 macOS，請透過 Homebrew 安裝 [OrbStack](https://orbstack.dev/)：

```bash
brew install orbstack
```

以最精簡的設定啟動 PostgreSQL：

```bash
docker run --name db -e POSTGRES_PASSWORD=password -p 5432:5432 -d postgres
```

- `docker run` 會建立並啟動新的 container。
- `--name db` 將 container 命名為 `db`，之後便能透過 `docker stop db`、`docker logs db` 等指令輕鬆引用它。
- `-e POSTGRES_PASSWORD=password` 將 PostgreSQL 預設管理員帳號 `postgres` 的密碼設為 `password`。
- `-p 5432:5432` 將 host 的 `5432` port 對應至 container 的 `5432` port。
- `-d` 讓 container 在背景執行。
- `postgres` 指定要使用的 Docker image。

若 Docker 在 Linux 虛擬機器內執行，請將連線設定中的 `localhost` 替換成該虛擬機器的 IP 位址。

確認 PostgreSQL 是否成功啟動：

```bash
docker logs db
```

之後可透過下列指令啟動或停止 container：

```bash
docker start db
docker stop db
```

### 使用 GoLand 連線至 PostgreSQL

1. 開啟 Database 工具視窗。

   ![](./pic/1.png)

2. 點選 `+`，選擇 `Data Source`，再選擇 `PostgreSQL`。

   ![](./pic/2.png)

3. 設定連線：

   - **Host：** 在 macOS 或 Docker 直接執行於 Linux 時使用 `localhost`。若 Docker 在虛擬機器中執行，請使用該虛擬機器的 IP 位址。
   - **User：** 使用 PostgreSQL 的預設使用者 `postgres`。
   - **Password：** 使用 `POSTGRES_PASSWORD` 指定的值；本設定中為 `password`。

   ![](./pic/3.png)

4. 點選 `Test Connection`。第一次嘗試連線時，系統可能會提示你下載 PostgreSQL driver。請安裝 driver 後再次測試連線。

   連線成功時會顯示下列訊息：

   ![](./pic/4.png)

5. 點選 `OK` 儲存設定。

   連線會出現在 Database 工具視窗中，你可以在此瀏覽及管理 PostgreSQL server 上的資料庫。每個資料庫彼此隔離，但共用同一個 PostgreSQL instance。

   ![](./pic/5.png)

### 安裝必要工具

#### sqlc

安裝 sqlc 命令列工具：

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

[sqlc](https://sqlc.dev/) 會根據 SQL 查詢與 annotation 產生型別安全的 Go 程式碼。例如：

```sql
-- name: GetMemberByID :one
SELECT id, name, email
FROM members
WHERE id = $1;
```

sqlc 會根據含有 annotation 的查詢產生參數型別、結果型別及查詢方法。這可減少重複的資料庫存取程式碼，並避免將欄位掃描至不相容 Go 值等常見錯誤。

安裝完成後，請參閱官方 [sqlc 指南](https://docs.sqlc.dev/en/latest/tutorials/getting-started-postgresql.html#schema-and-queries)以了解更多資訊。

#### golang-migrate

將 golang-migrate 加入專案：

```bash
go get github.com/golang-migrate/migrate/v4@v4.19.0
```

[golang-migrate](https://github.com/golang-migrate/migrate) 透過依序排列且具有版本編號的 migration 管理資料庫綱要變更。每項變更都有一個套用變更的 `up` migration，以及一個還原變更的 `down` migration：

```text
000001_create_members.up.sql
000001_create_members.down.sql
000002_create_courses.up.sql
000002_create_courses.down.sql
```

Migration 檔名遵循 `<version>_<description>.up.sql` 與 `<version>_<description>.down.sql` 格式。同一組的兩個檔案使用相同的版本與描述：

- `.up.sql` 檔會套用變更，例如建立或修改資料表。
- 對應的 `.down.sql` 檔會還原該變更，例如刪除資料表或恢復原本的結構。
- Rollback 會按照版本的反向順序執行。因此，有相依關係的物件必須先於其引用的物件移除；在本實作中，應先刪除 `enrollments`，再刪除 `members` 或 `courses`。

例如，下列 down migration 會還原建立 `members` 資料表的對應 up migration：

```sql
-- 000001_create_members.down.sql
DROP TABLE IF EXISTS members;
```

Migration library 不會自行執行。提供的整合測試會先呼叫 `databaseutil/migration.go` 中的 helper，再執行 assertion，因此 `go test ./internal/...` 會自動檢查目前的綱要版本並套用尚未執行的 up migration。它不會執行 down migration，也不會刪除現有資料。

#### Go 套件

安裝用於 logging、UUID 及 PostgreSQL 存取的套件：

```bash
go get go.uber.org/zap
go get github.com/google/uuid
go get github.com/jackc/pgx/v5
go get github.com/jackc/pgx/v5/pgxpool
```

## 專案架構

以下目錄樹列出本實作練習的重要檔案：

- `[TODO]`：作業中需要建立或編輯的檔案。
- `[TEST]`：提供的評測程式碼，請勿編輯。
- `[GENERATED]`：由 script 或 sqlc 產生，請勿手動編輯。
- 沒有標籤的檔案是專案提供的支援檔案。

```text
backend-database-lab/
|-- databaseutil/
|   `-- migration.go                    # 為測試套用尚未執行的 migration
|-- internal/
|   |-- database/
|   |   |-- migrations/                 # [TODO] 在此新增有編號的 up/down migration 組合
|   |   |   |-- 000001_example.up.sql
|   |   |   `-- 000001_example.down.sql
|   |   `-- full_schema.sql             # [GENERATED] 合併各 domain 的 schema
|   |-- member/
|   |   |-- schema.sql                  # [TODO] 定義 members 資料表
|   |   |-- queries.sql                 # [TODO] 實作 member 查詢
|   |   |-- queries_test.go             # [TEST] 執行 DB migration 並評測 members
|   |   |-- db.go                       # [GENERATED] sqlc 資料庫介面
|   |   |-- models.go                   # [GENERATED] sqlc 資料庫 model
|   |   `-- queries.sql.go              # [GENERATED] Member 查詢方法
|   |-- course/
|   |   |-- schema.sql                  # [TODO] 定義 courses 資料表
|   |   |-- queries.sql                 # [TODO] 實作 course 查詢
|   |   |-- queries_test.go             # [TEST] 執行 DB migration 並評測 courses
|   |   |-- db.go                       # [GENERATED] sqlc 資料庫介面
|   |   |-- models.go                   # [GENERATED] sqlc 資料庫 model
|   |   `-- queries.sql.go              # [GENERATED] Course 查詢方法
|   `-- enrollment/
|       |-- schema.sql                  # [TODO] 定義 enrollments 資料表
|       |-- queries.sql                 # [TODO] 實作 enrollment 查詢
|       |-- queries_test.go             # [TEST] 執行 DB migration 並評測 enrollments
|       |-- db.go                       # [GENERATED] sqlc 資料庫介面
|       |-- models.go                   # [GENERATED] sqlc 資料庫 model
|       `-- queries.sql.go              # [GENERATED] Enrollment 查詢方法
|-- scripts/
|   `-- create_sqlc_full_schema.sh      # 產生 full_schema.sql 與 sqlc.yaml
|-- sqlc.yaml                           # [GENERATED] sqlc 設定檔
|-- go.mod                              # Go module 與 dependency 宣告
|-- go.sum                              # Dependency checksum
`-- README.md                           # 實作說明與規格契約
```

請依照下列順序完成作業：

1. 在各 domain 的 `schema.sql` 中定義資料表，並在 `internal/database/migrations/` 下新增對應的 migration 組合。
2. 在各 domain 的 `queries.sql` 中實作含有 annotation 的 SQL 查詢。
3. 執行 `./scripts/create_sqlc_full_schema.sh`，產生 `internal/database/full_schema.sql` 與 `sqlc.yaml`。
4. 執行 `sqlc generate`，在各 domain 中產生 Go 查詢層。
5. 執行 `go test ./internal/...`，自動套用尚未執行的 migration，並評測產生的查詢方法與資料庫約束。

只能編輯標有 `[TODO]` 的檔案。請勿修改提供的 `queries_test.go`，也不要手動編輯 `full_schema.sql`、`sqlc.yaml` 或任何由 sqlc 產生的 Go 檔案。

## 第一部分：設計資料表

為選課系統設計並實作關聯式資料庫綱要。綱要至少必須符合第三正規化（3NF），並包含以下資料表。

### `schema.sql` 應包含哪些內容？

資料庫綱要是資料庫的藍圖，描述資料表、欄位、PostgreSQL 資料型別，以及 `PRIMARY KEY`、`FOREIGN KEY`、`NOT NULL`、`UNIQUE`、`CHECK` 和 `DEFAULT` 等約束。`schema.sql` 檔案包含這些結構定義，不應包含 `SELECT`、`INSERT`、`UPDATE` 或 `DELETE` 等應用程式查詢，也不應包含範例資料。

請以下列形式為參考，並依據下方的「綱要規格契約」替換預留內容：

```sql
CREATE TABLE table_name (
    column_name DATA_TYPE CONSTRAINTS,
    ...
);
```

請將每個 domain 的資料表定義分別放在各自的檔案中：

- `internal/member/schema.sql` 定義 `members` 資料表。
- `internal/course/schema.sql` 定義 `courses` 資料表。
- `internal/enrollment/schema.sql` 定義 `enrollments` 資料表。

`scripts/create_sqlc_full_schema.sh` script 會合併這些檔案，讓 sqlc 得知完整的資料庫結構，並在產生 Go 程式碼前驗證各 `queries.sql` 檔案中的 SQL。

雖然 schema 檔與 migration 都包含 DDL，但兩者用途不同：

| | `schema.sql` | Migration 檔案 |
| --- | --- | --- |
| **用途** | 完整且最新的資料庫結構快照 | 資料庫結構依序變更的歷史紀錄 |
| **使用者** | sqlc（在 `scripts/create_sqlc_full_schema.sh` 合併各 domain 檔案後使用） | golang-migrate（依版本順序執行檔案） |
| **效果** | 協助 sqlc 理解並驗證查詢，不會變更 PostgreSQL 資料庫 | 實際在 PostgreSQL 資料庫中建立、變更或移除物件 |
| **組織方式** | 每個 domain 一個檔案 | 有編號的 `up.sql` 與 `down.sql` 成對檔案 |

新增或變更資料表時，請更新該 domain 的 `schema.sql`，並為同一項變更新增 migration。只更新 `schema.sql` 會讓 sqlc 看見實際資料庫並不存在的結構；只新增 migration 則會讓 sqlc 使用過時的描述。

所有 `up` migration 執行完畢後，實際資料庫結構必須與合併後的 `schema.sql` 檔案一致。

### 綱要規格契約

#### `members`

| 欄位 | PostgreSQL 型別 | 要求 |
| --- | --- | --- |
| `id` | `UUID` | Primary key；預設值為 `gen_random_uuid()` |
| `name` | `TEXT` | 必填 |
| `email` | `TEXT` | 必填且不可重複 |
| `joined_at` | `TIMESTAMPTZ` | 必填；預設值為 `now()` |

#### `courses`

| 欄位 | PostgreSQL 型別 | 要求 |
| --- | --- | --- |
| `id` | `UUID` | Primary key；預設值為 `gen_random_uuid()` |
| `title` | `TEXT` | 必填 |
| `capacity` | `INTEGER` | 必填且必須大於零 |
| `created_at` | `TIMESTAMPTZ` | 必填；預設值為 `now()` |

#### `enrollments`

| 欄位 | PostgreSQL 型別 | 要求 |
| --- | --- | --- |
| `member_id` | `UUID` | 參照 `members(id)`，並設為 `ON DELETE CASCADE` |
| `course_id` | `UUID` | 參照 `courses(id)`，並設為 `ON DELETE CASCADE` |
| `status` | `TEXT` | 必填；預設值為 `enrolled` |
| `enrolled_at` | `TIMESTAMPTZ` | 必填；預設值為 `now()` |

`(member_id, course_id)` 這組欄位必須作為 `enrollments` 的 primary key，以防止同一名成員重複選修同一門課程。

`status` 欄位只能接受下列值：

- `enrolled`
- `completed`
- `cancelled`

三個資料表中的所有欄位皆須設定為 `NOT NULL`。

### Goals

1. 建立 ERD，呈現 entity、primary key、foreign key、relationship 與 cardinality。
2. 在 `member`、`course` 與 `enrollment` 各 domain 的 `schema.sql` 中撰寫資料表 DDL。這些檔案會由 `scripts/create_sqlc_full_schema.sh` 使用。
3. 為完整綱要撰寫成對的 `up` 與 `down` migration 檔案。請使用 `000001`、`000002`、`000003` 等固定寬度的序號，讓 golang-migrate 與 sqlc 以相同順序處理檔案。
4. 確保合併後的 `schema.sql` 檔案描述的綱要，與套用所有 `up` migration 後產生的最終綱要一致。
5. 實作所有必要的 `PRIMARY KEY`、`FOREIGN KEY`、`NOT NULL`、`UNIQUE`、`CHECK` 與 `DEFAULT` 約束，但不得直接從本規格複製完成的 `CREATE TABLE` statement。

## 第二部分：SQL 查詢與 Queries Layer

為 `member`、`course` 與 `enrollment` domain 建立型別安全的查詢層。每個 domain 必須包含：

- `schema.sql`
- `queries.sql`

請勿手動建立或編輯 `sqlc.yaml`，而應使用現有 script 產生：

```bash
./scripts/create_sqlc_full_schema.sh
sqlc generate
```

請在各 domain 的 `queries.sql` 中撰寫所有含 annotation 的查詢。以下查詢名稱、參數及產生的回傳型別，是提供給 service 與 handler 使用的固定契約。

凡是包含多個參數的查詢，請使用 [`sqlc.arg(...)`](https://docs.sqlc.dev/en/latest/howto/named_parameters.html)，不要使用 `$1`、`$2`、`$3` 等位置參數。具名參數會控制 sqlc 產生的 Go 參數 struct 欄位名稱。例如，`sqlc.arg(member_id)` 會產生 `MemberID` 欄位，而非推測或依位置命名的欄位。

下方列出的產生參數型別是 sqlc 所需輸出的說明，請勿將它們複製到 Go 檔案中。

所有 domain 均遵循下列 CRUD 規則：

- 查詢 annotation、名稱與 cardinality 必須符合下方契約。
- Create、update 與 delete 查詢必須使用 `:one` 和 `RETURNING` 回傳受影響的資料列。
- 每個 `SELECT` 與 `RETURNING` clause 都必須明確列出欄位，不得使用 `*`。
- 每個 update 與 delete 都必須包含完整的 `WHERE` clause，精確指定單一資源。
- List 查詢不使用 pagination，且必須包含下方指定的 deterministic `ORDER BY`。

### Member 查詢契約

請在 `internal/member/queries.sql` 實作下列含 annotation 的查詢：

| 查詢 | 輸入 | 必要行為 |
| --- | --- | --- |
| `CreateMember :one` | `name`、`email` | 新增 member，並回傳 `id`、`name`、`email` 與 `joined_at`。 |
| `GetMember :one` | `id` | 回傳 `id` 指定的 member；若不存在則回傳 `pgx.ErrNoRows`。 |
| `ListMembers :many` | 無 | 回傳所有 member，並依 `joined_at ASC, id ASC` 排序。 |
| `UpdateMember :one` | `name`、`email`、`id` | 完整取代 `id` 指定之 member 的 `name` 與 `email`，再回傳該資料列。 |
| `DeleteMember :one` | `id` | 僅刪除 `id` 指定的 member，並回傳被刪除的資料列。 |
| `ListMemberCourses :many` | `member_id` | 依下方說明回傳該 member 的完整選課紀錄。 |
| `ListClassmates :many` | `member_id` | 依下方說明回傳不重複的目前同學名單。 |

執行 `sqlc generate` 後，必須產生具有下列欄位的參數型別：

```go
type CreateMemberParams struct {
    Name  string
    Email string
}

type UpdateMemberParams struct {
    Name  string
    Email string
    ID    uuid.UUID
}
```

#### JOIN：列出 Member 的課程

使用 `members`、`enrollments` 與 `courses` 實作 `ListMemberCourses`。

此查詢必須：

- 接受一個 member ID。
- 包含該 member 的完整選課紀錄，不受 status 限制。
- 僅回傳下列欄位與 alias：

| Alias | PostgreSQL 型別 |
| --- | --- |
| `course_id` | `UUID` |
| `course_title` | `TEXT` |
| `status` | `TEXT` |
| `enrolled_at` | `TIMESTAMPTZ` |

- 依 `course_title ASC, course_id ASC` 排序。

產生的結果型別必須為：

```go
type ListMemberCoursesRow struct {
    CourseID    uuid.UUID
    CourseTitle string
    Status      string
    EnrolledAt  time.Time
}
```

#### JOIN：列出同學

使用 `enrollments` 的 self-join 以及與 `members` 的 join 實作 `ListClassmates`。

此查詢必須：

- 接受一個 member ID。
- 找出與指定 member 至少共同選修一門課程的其他 member。
- 指定 member 與同學的 enrollment status 都必須是 `enrolled`。
- 從結果中排除指定的 member。
- 使用 `DISTINCT`，讓共同選修多門課程的同學只出現一次。
- 僅回傳下列欄位與 alias：

| Alias | PostgreSQL 型別 |
| --- | --- |
| `member_id` | `UUID` |
| `member_name` | `TEXT` |
| `member_email` | `TEXT` |

- 依 `member_name ASC, member_id ASC` 排序。

產生的結果型別必須為：

```go
type ListClassmatesRow struct {
    MemberID    uuid.UUID
    MemberName  string
    MemberEmail string
}
```

### Course 查詢契約

請在 `internal/course/queries.sql` 實作下列含 annotation 的查詢：

| 查詢 | 輸入 | 必要行為 |
| --- | --- | --- |
| `CreateCourse :one` | `title`、`capacity` | 新增 course，並回傳 `id`、`title`、`capacity` 與 `created_at`。 |
| `GetCourse :one` | `id` | 回傳 `id` 指定的 course；若不存在則回傳 `pgx.ErrNoRows`。 |
| `ListCourses :many` | 無 | 回傳所有 course，並依 `created_at ASC, id ASC` 排序。 |
| `UpdateCourse :one` | `title`、`capacity`、`id` | 完整取代 `id` 指定之 course 的 `title` 與 `capacity`，再回傳該資料列。 |
| `DeleteCourse :one` | `id` | 僅刪除 `id` 指定的 course，並回傳被刪除的資料列。 |
| `ListCourseRoster :many` | `course_id` | 依下方說明回傳該課程目前有效的 enrollment。 |

執行 `sqlc generate` 後，必須產生具有下列欄位的參數型別：

```go
type CreateCourseParams struct {
    Title    string
    Capacity int32
}

type UpdateCourseParams struct {
    Title    string
    Capacity int32
    ID       uuid.UUID
}
```

#### JOIN：列出課程名單

使用 `courses`、`enrollments` 與 `members` 實作 `ListCourseRoster`。

此查詢必須：

- 接受一個 course ID。
- 僅包含 status 為 `enrolled` 的 enrollment。
- 排除 `completed` 與 `cancelled` enrollment。
- 僅回傳下列欄位與 alias：

| Alias | PostgreSQL 型別 |
| --- | --- |
| `member_id` | `UUID` |
| `member_name` | `TEXT` |
| `member_email` | `TEXT` |
| `status` | `TEXT` |
| `enrolled_at` | `TIMESTAMPTZ` |

- 依 `member_name ASC, member_id ASC` 排序。

產生的結果型別必須為：

```go
type ListCourseRosterRow struct {
    MemberID    uuid.UUID
    MemberName  string
    MemberEmail string
    Status      string
    EnrolledAt  time.Time
}
```

### Enrollment 查詢契約

請在 `internal/enrollment/queries.sql` 實作下列含 annotation 的查詢：

| 查詢 | 輸入 | 必要行為 |
| --- | --- | --- |
| `CreateEnrollment :one` | `member_id`、`course_id` | 使用預設的 `enrolled` status 新增 enrollment，並回傳建立的資料列。 |
| `GetEnrollment :one` | `member_id`、`course_id` | 回傳由 member/course composite key 指定的 enrollment；若不存在則回傳 `pgx.ErrNoRows`。 |
| `ListEnrollments :many` | 無 | 回傳所有 enrollment，並依 `enrolled_at ASC, member_id ASC, course_id ASC` 排序。 |
| `UpdateEnrollmentStatus :one` | `status`、`member_id`、`course_id` | 僅更新由 composite key 指定之 enrollment 的 `status`，再回傳該資料列。 |
| `DeleteEnrollment :one` | `member_id`、`course_id` | 僅刪除由 composite key 指定的 enrollment，並回傳被刪除的資料列。 |
| `GetEnrollmentDetail :one` | `member_id`、`course_id` | 依下方說明回傳 join 後的 enrollment 詳細資料。 |

執行 `sqlc generate` 後，必須產生具有下列欄位的參數型別：

```go
type CreateEnrollmentParams struct {
    MemberID uuid.UUID
    CourseID uuid.UUID
}

type GetEnrollmentParams struct {
    MemberID uuid.UUID
    CourseID uuid.UUID
}

type UpdateEnrollmentStatusParams struct {
    Status   string
    MemberID uuid.UUID
    CourseID uuid.UUID
}

type DeleteEnrollmentParams struct {
    MemberID uuid.UUID
    CourseID uuid.UUID
}

type GetEnrollmentDetailParams struct {
    MemberID uuid.UUID
    CourseID uuid.UUID
}
```

#### JOIN：取得 Enrollment 詳細資料

使用全部三個資料表實作 `GetEnrollmentDetail`。

此查詢必須：

- 透過 `sqlc.arg(...)` 接受 `member_id` 與 `course_id`。
- 使用 `:one` annotation。
- 當 enrollment 不存在時，透過產生的方法回傳 `pgx.ErrNoRows`。
- 僅回傳下列欄位與 alias：

| Alias | PostgreSQL 型別 |
| --- | --- |
| `member_id` | `UUID` |
| `member_name` | `TEXT` |
| `member_email` | `TEXT` |
| `course_id` | `UUID` |
| `course_title` | `TEXT` |
| `course_capacity` | `INTEGER` |
| `status` | `TEXT` |
| `enrolled_at` | `TIMESTAMPTZ` |

產生的結果型別必須為：

```go
type GetEnrollmentDetailRow struct {
    MemberID       uuid.UUID
    MemberName     string
    MemberEmail    string
    CourseID       uuid.UUID
    CourseTitle    string
    CourseCapacity int32
    Status         string
    EnrolledAt     time.Time
}
```

### Goals

1. 完成三個 domain 的 `schema.sql` 檔案及所有成對的 migration。
2. 為每個 domain 新增含 annotation 的 `queries.sql` 檔案。
3. 執行 `./scripts/create_sqlc_full_schema.sh`，建立完整綱要與 `sqlc.yaml`。
4. 執行 `sqlc generate`。
5. 提交由 `sqlc generate` 產生的 Go 檔案。請勿自行建立或手動編輯產生的 Go 程式碼。
6. 撰寫測試或小型 Go 程式，呼叫產生的查詢方法。

### 驗證

1. `sqlc generate` 能順利完成且沒有錯誤。
2. `go test ./...` 能編譯並測試產生的查詢方法。
3. CRUD 操作會回傳預期的資料列，且僅影響指定目標。
4. 沒有 enrollment 的 member 從 `ListMemberCourses` 取得的結果不包含任何資料列。
5. 沒有有效 enrollment 的 course 從 `ListCourseRoster` 取得的結果不包含任何資料列。
6. `completed` 與 `cancelled` enrollment 不會出現在 `ListCourseRoster` 中。
7. 在多門課程中都是同學的 member，只會在 `ListClassmates` 中出現一次。
8. enrollment 不存在時，`GetEnrollmentDetail` 會回傳 `pgx.ErrNoRows`。
9. 重複或無效的 enrollment 會被第一部分實作的約束拒絕。

## 評測

本實作採通過／不通過制。

### 執行評測前

1. 使用本實作全程採用的連線設定啟動 PostgreSQL：

   ```text
   postgresql://postgres:password@localhost:5432/postgres?sslmode=disable
   ```

2. 產生 sqlc 設定與 Go 程式碼：

   ```bash
   ./scripts/create_sqlc_full_schema.sh
   sqlc generate
   ```

3. 執行評測。測試會在檢查綱要與查詢行為前，自動套用所有尚未執行的 up migration：

   ```bash
   go test ./internal/...
   ```

### 通過標準

提交內容符合下列所有條件即為通過：

- sqlc 產生的 package 能以規定的查詢名稱、參數及結果欄位成功編譯。
- 自動 migration 順利完成，且資料庫 preflight 能找到 `members`、`courses` 與 `enrollments` 資料表。
- 10 項 member 測試、10 項 course 測試及 14 項 enrollment 測試全數通過。
- 提供的 `queries_test.go` 及產生的檔案未經手動修改。

### Member 評測（10 項測試）

| 項目 | 評測行為 |
| --- | --- |
| CRUD（5） | Create、get、list、update 及 delete 會回傳預期的 member 資料，且僅影響指定的資料列。 |
| 錯誤與約束（2） | member 不存在時回傳 `pgx.ErrNoRows`，且重複 email 會被 unique constraint 拒絕。 |
| Member 課程（2） | 完整選課紀錄包含每一種 status、遵循規定的排序，並能在適當情況下回傳空 list。 |
| 同學（1） | 結果會依有效 enrollment 篩選、排除指定 member、移除重複項目，並遵循規定的排序。 |

### Course 評測（10 項測試）

| 項目 | 評測行為 |
| --- | --- |
| CRUD（5） | Create、get、list、update 及 delete 會回傳預期的 course 資料，且僅影響指定的資料列。 |
| 錯誤與約束（3） | course 不存在時回傳 `pgx.ErrNoRows`；零與負數的 capacity 會被 check constraint 拒絕。 |
| 課程名單（2） | 名單僅包含有效 enrollment，排除 completed 與 cancelled enrollment，遵循規定的排序，且可為空。 |

### Enrollment 評測（14 項測試）

| 項目 | 評測行為 |
| --- | --- |
| CRUD（5） | Create、get、list、update status 及 delete 使用 member/course composite identifier，並回傳預期資料。 |
| 錯誤與約束（5） | 重複組合、不存在的 member/course reference 及無效 status 會被拒絕；enrollment 不存在時回傳 `pgx.ErrNoRows`。 |
| Enrollment 詳細資料（2） | Detail 查詢會回傳規定的 member、course 與 enrollment join 欄位；若不存在則回傳 `pgx.ErrNoRows`。 |
| Cascade（2） | 刪除相關的 member 或 course 時，會自動移除 enrollment。 |

如果產生的 package 無法編譯，請先依契約檢查查詢 annotation、名稱、參數、選取的欄位與 alias。啟動失敗表示 PostgreSQL 無法使用、migration 檔案遺失或無效，或 migration 未建立全部三個資料表。Assertion 失敗或出現非預期的 SQLSTATE，表示查詢行為或資料庫約束不符合規格。

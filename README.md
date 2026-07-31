# Backend Database Lab

This lab is part of the NYCU Software Development Club (SDC) backend training program and focuses on the database layer of a course enrollment system.

By completing it, you will learn to design a normalized relational schema and describe its current structure with `schema.sql` while evolving it through versioned migrations.

You will also use sqlc to implement type-safe CRUD and JOIN queries, then verify query results and database constraints with PostgreSQL integration tests.

## How to Start

### Fork and Clone the Repository

1. Open the [original repository](https://github.com/ilsao/backend-database-lab).
2. Click `Fork` in the upper-right corner, then click `Create fork` to copy the repository to your GitHub account.
3. Clone your fork and enter the project directory. Replace `<your-github-username>` with your GitHub username:

   ```bash
   git clone https://github.com/<your-github-username>/backend-database-lab.git
   cd backend-database-lab
   ```

Make sure you clone your own fork rather than the original repository. This allows you to push your work to your own GitHub repository.

If you enjoy this lab, you can also return to the [original repository](https://github.com/ilsao/backend-database-lab) and click `Star`.

### Set Up Docker

If you use Linux or WSL, run the following commands to install Docker:

```bash
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER
```

If you use macOS, install [OrbStack](https://orbstack.dev/) with Homebrew:

```bash
brew install orbstack
```

Start PostgreSQL with a minimal configuration:

```bash
docker run --name db -e POSTGRES_PASSWORD=password -p 5432:5432 -d postgres
```

- `docker run` creates and starts a new container.
- `--name db` names the container `db`, making it easier to reference later with commands such as `docker stop db` and `docker logs db`.
- `-e POSTGRES_PASSWORD=password` sets the password for PostgreSQL's default administrator account, `postgres`.
- `-p 5432:5432` maps port `5432` on the host to port `5432` in the container.
- `-d` runs the container in the background.
- `postgres` specifies the Docker image.

If Docker is running inside a Linux virtual machine, replace `localhost` in the connection settings with the virtual machine's IP address.

Check whether PostgreSQL started successfully:

```bash
docker logs db
```

You can later start or stop the container with:

```bash
docker start db
docker stop db
```

### Connect to PostgreSQL with GoLand

1. Open the Database tool window.

   ![](./pic/1.png)

2. Click `+`, select `Data Source`, and then select `PostgreSQL`.

   ![](./pic/2.png)

3. Configure the connection:

   - **Host:** Use `localhost` on macOS or when Docker runs directly on Linux. When Docker runs in a virtual machine, use the virtual machine's IP address.
   - **User:** Use PostgreSQL's default user, `postgres`.
   - **Password:** Use the value specified by `POSTGRES_PASSWORD`, which is `password` in this setup.

   ![](./pic/3.png)

4. Click `Test Connection`. The first connection attempt may prompt you to download the PostgreSQL driver. Install it and test the connection again.

   A successful connection displays the following message:

   ![](./pic/4.png)

5. Click `OK` to save the configuration.

    The connection appears in the Database tool window, where you can browse and manage the databases hosted by the PostgreSQL server. Each database is isolated while sharing the same PostgreSQL instance.

    ![](./pic/5.png)

### Install the Required Tools

#### sqlc

Install the sqlc command-line tool:

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

[sqlc](https://sqlc.dev/) generates type-safe Go code from SQL queries and annotations. For example:

```sql
-- name: GetMemberByID :one
SELECT id, name, email
FROM members
WHERE id = $1;
```

From an annotated query, sqlc generates parameter types, result types, and a query method. This reduces repetitive database access code and prevents common mistakes such as scanning columns into incompatible Go values.

After installation, refer to the official [sqlc guide](https://docs.sqlc.dev/en/latest/tutorials/getting-started-postgresql.html#schema-and-queries) for further imformation.

#### golang-migrate

Add golang-migrate to the project:

```bash
go get github.com/golang-migrate/migrate/v4@v4.19.0
```

[golang-migrate](https://github.com/golang-migrate/migrate) manages database schema changes as ordered, versioned migrations. Each change has an `up` migration that applies it and a `down` migration that reverses it:

```text
000001_create_members.up.sql
000001_create_members.down.sql
000002_create_courses.up.sql
000002_create_courses.down.sql
```

The migration library does not run automatically. The backend must explicitly call a migration function, such as the helper in `databaseutil/migration.go`, during startup. The helper then checks the current schema version and applies pending migrations.

#### Go Packages

Install the packages used for logging, UUIDs, and PostgreSQL access:

```bash
go get go.uber.org/zap
go get github.com/google/uuid
go get github.com/jackc/pgx/v5
go get github.com/jackc/pgx/v5/pgxpool
```

## Architecture

The following tree shows the files that are important for this lab:

- `[TODO]`: Create or edit this file as part of the assignment.
- `[TEST]`: Provided evaluation code; do not edit it.
- `[GENERATED]`: Created by a script or sqlc; do not edit it manually.
- Files without a label are provided project support files.

```text
backend-database-lab/
|-- cmd/
|   `-- main.go                         # Starts the application and runs migrations
|-- databaseutil/
|   `-- migration.go                    # Helper for applying pending migrations
|-- internal/
|   |-- database/
|   |   |-- migrations/                 # [TODO] Add numbered up/down migration pairs here
|   |   |   |-- 000001_example.up.sql
|   |   |   `-- 000001_example.down.sql
|   |   `-- full_schema.sql             # [GENERATED] Combined domain schemas
|   |-- member/
|   |   |-- schema.sql                  # [TODO] Define the members table
|   |   |-- queries.sql                 # [TODO] Implement member queries
|   |   |-- queries_test.go             # [TEST] Member evaluation tests
|   |   |-- db.go                       # [GENERATED] sqlc database interface
|   |   |-- models.go                   # [GENERATED] sqlc database models
|   |   `-- queries.sql.go              # [GENERATED] Member query methods
|   |-- course/
|   |   |-- schema.sql                  # [TODO] Define the courses table
|   |   |-- queries.sql                 # [TODO] Implement course queries
|   |   |-- queries_test.go             # [TEST] Course evaluation tests
|   |   |-- db.go                       # [GENERATED] sqlc database interface
|   |   |-- models.go                   # [GENERATED] sqlc database models
|   |   `-- queries.sql.go              # [GENERATED] Course query methods
|   `-- enrollment/
|       |-- schema.sql                  # [TODO] Define the enrollments table
|       |-- queries.sql                 # [TODO] Implement enrollment queries
|       |-- queries_test.go             # [TEST] Enrollment evaluation tests
|       |-- db.go                       # [GENERATED] sqlc database interface
|       |-- models.go                   # [GENERATED] sqlc database models
|       `-- queries.sql.go              # [GENERATED] Enrollment query methods
|-- scripts/
|   `-- create_sqlc_full_schema.sh      # Generates full_schema.sql and sqlc.yaml
|-- sqlc.yaml                           # [GENERATED] sqlc configuration
|-- go.mod                              # Go module and dependency declarations
|-- go.sum                              # Dependency checksums
`-- README.md                           # Lab instructions and contracts
```

Complete the assignment in this order:

1. Define each domain's table in its `schema.sql` and add matching migration pairs under `internal/database/migrations/`.
2. Implement the annotated SQL queries in each domain's `queries.sql`.
3. Run `./scripts/create_sqlc_full_schema.sh` to generate `internal/database/full_schema.sql` and `sqlc.yaml`.
4. Run `sqlc generate` to generate the Go query layer inside each domain.
5. Run `go test ./internal/...` to evaluate the generated query methods and database constraints.

Only edit files marked `[TODO]`. Do not modify the provided `queries_test.go` files or manually edit `full_schema.sql`, `sqlc.yaml`, or any Go files generated by sqlc.

## Part I: Design Tables

Design and implement the relational schema for a course enrollment system. The schema must be normalized to at least Third Normal Form (3NF) and must contain the following tables.

### What Goes in `schema.sql`?

A database schema is the blueprint of the database. It describes tables, columns, PostgreSQL data types, and constraints such as `PRIMARY KEY`, `FOREIGN KEY`, `NOT NULL`, `UNIQUE`, `CHECK`, and `DEFAULT`. A `schema.sql` file contains these structure definitions. It does not contain application queries such as `SELECT`, `INSERT`, `UPDATE`, or `DELETE`, and it should not contain sample data.

Use the following shape as a guide and replace the placeholders with the requirements in the Schema Contract below:

```sql
CREATE TABLE table_name (
    column_name DATA_TYPE CONSTRAINTS,
    ...
);
```

Keep each domain's table definition in its own file:

- `internal/member/schema.sql` defines the `members` table.
- `internal/course/schema.sql` defines the `courses` table.
- `internal/enrollment/schema.sql` defines the `enrollments` table.

The `scripts/create_sqlc_full_schema.sh` script combines these files so sqlc knows the complete database structure and can validate the SQL in each `queries.sql` file before generating Go code.

Although both contain DDL, the schema files and migrations serve different purposes:

| | `schema.sql` | Migration files |
| --- | --- | --- |
| **Purpose** | A snapshot of the complete current database structure | An ordered history of changes to the database structure |
| **Used by** | sqlc, after `scripts/create_sqlc_full_schema.sh` combines the domain files | golang-migrate, which runs the files in version order |
| **Effect** | Helps sqlc understand and validate queries; it does not change the PostgreSQL database | Actually creates, changes, or removes objects in the PostgreSQL database |
| **Organization** | One file for each domain | Numbered pairs of `up.sql` and `down.sql` files |

When you add or change a table, update its domain's `schema.sql` and add a new migration for the same change. Updating only `schema.sql` would let sqlc see a structure that the real database does not have, while adding only a migration would leave sqlc with an outdated description.

After all `up` migrations run, the actual database structure must match the combined `schema.sql` files.

### Schema Contract

#### `members`

| Column | PostgreSQL Type | Requirements |
| --- | --- | --- |
| `id` | `UUID` | Primary key; defaults to `gen_random_uuid()` |
| `name` | `TEXT` | Required |
| `email` | `TEXT` | Required and unique |
| `joined_at` | `TIMESTAMPTZ` | Required; defaults to `now()` |

#### `courses`

| Column | PostgreSQL Type | Requirements |
| --- | --- | --- |
| `id` | `UUID` | Primary key; defaults to `gen_random_uuid()` |
| `title` | `TEXT` | Required |
| `capacity` | `INTEGER` | Required and greater than zero |
| `created_at` | `TIMESTAMPTZ` | Required; defaults to `now()` |

#### `enrollments`

| Column | PostgreSQL Type | Requirements |
| --- | --- | --- |
| `member_id` | `UUID` | References `members(id)` with `ON DELETE CASCADE` |
| `course_id` | `UUID` | References `courses(id)` with `ON DELETE CASCADE` |
| `status` | `TEXT` | Required; defaults to `enrolled` |
| `enrolled_at` | `TIMESTAMPTZ` | Required; defaults to `now()` |

The pair `(member_id, course_id)` must be the primary key of `enrollments`. This prevents a member from enrolling in the same course more than once.

The `status` column must accept only the following values:

- `enrolled`
- `completed`
- `cancelled`

All columns in the three tables must be `NOT NULL`.

### Deliverables

1. Create an ERD that shows the entities, primary keys, foreign keys, relationships, and cardinalities.
2. Write the table DDL in the `schema.sql` file for each of the `member`, `course`, and `enrollment` domains. These files are used by `scripts/create_sqlc_full_schema.sh`.
3. Write paired `up` and `down` migration files for the complete schema. Use fixed-width sequence numbers such as `000001`, `000002`, and `000003` so golang-migrate and sqlc process the files in the same order.
4. Ensure that the combined `schema.sql` files describe the same final schema produced by applying every `up` migration.
5. Implement all required `PRIMARY KEY`, `FOREIGN KEY`, `NOT NULL`, `UNIQUE`, `CHECK`, and `DEFAULT` constraints without copying a completed `CREATE TABLE` statement from this specification.

### Verification

Demonstrate that:

1. Applying all `up` migrations to an empty database creates the expected tables and constraints.
2. Invalid data is rejected, including duplicate member emails, non-positive course capacity, unsupported enrollment status, enrollment with missing references, and duplicate enrollment.
3. Applying all `down` migrations removes the schema without manual cleanup.
4. Applying the `up` migrations again after rollback succeeds.
5. The combined `schema.sql` files and the migrated database expose the same tables, columns, types, and constraints.

## Part II: SQL Query and Queries Layer

Build type-safe query layers for the `member`, `course`, and `enrollment` domains. Each domain must contain:

- `schema.sql`
- `queries.sql`

Do not create or edit `sqlc.yaml` manually. Generate it with the existing script:

```bash
./scripts/create_sqlc_full_schema.sh
sqlc generate
```

Write all annotated queries in each domain's `queries.sql`. The query names, parameters, and generated return types below are fixed contracts for the provided services and handlers.

### Member Queries Contract

Implement these annotated queries in `member/queries.sql`:

- `CreateMember :one`
- `GetMember :one`
- `ListMembers :many`
- `UpdateMember :one`
- `DeleteMember :one`
- `ListMemberCourses :many`
- `ListClassmates :many`

Running `sqlc generate` must produce parameter types with these fields:

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

### Course Queries Contract

Implement these annotated queries in `course/queries.sql`:

- `CreateCourse :one`
- `GetCourse :one`
- `ListCourses :many`
- `UpdateCourse :one`
- `DeleteCourse :one`
- `ListCourseRoster :many`

Running `sqlc generate` must produce parameter types with these fields:

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

### Enrollment Queries Contract

Implement these annotated queries in `enrollment/queries.sql`:

- `CreateEnrollment :one`
- `GetEnrollment :one`
- `ListEnrollments :many`
- `UpdateEnrollmentStatus :one`
- `DeleteEnrollment :one`
- `GetEnrollmentDetail :one`

Running `sqlc generate` must produce parameter types with these fields:

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

The code above documents generated types and must not be copied into a Go file. Use [`sqlc.arg(...)`](https://docs.sqlc.dev/en/latest/howto/named_parameters.html) for every parameter in a multi-parameter query so sqlc generates the exact field names shown above.

### CRUD Requirements

Implement the following CRUD behavior:

- Create, retrieve, list, fully update, and delete members.
- Create, retrieve, list, fully update, and delete courses.
- Create, retrieve, and list enrollments.
- Update an enrollment's status.
- Delete an enrollment.

Use these rules:

- Query annotations and names must match the contracts listed above.
- Create, update, and delete queries must use `:one` and `RETURNING` to return the affected row.
- Every `SELECT` and `RETURNING` clause must list its columns explicitly. Do not use `*`.
- Every update and delete must contain a complete `WHERE` clause that identifies one resource.
- List queries do not use pagination and must include a deterministic `ORDER BY`.

### JOIN Exercise 1: List a Member's Courses

Implement `ListMemberCourses` using `members`, `enrollments`, and `courses`.

The query must:

- Accept one member ID.
- Include the member's complete enrollment history, regardless of status.
- Return exactly these columns and aliases:

| Alias | PostgreSQL Type |
| --- | --- |
| `course_id` | `UUID` |
| `course_title` | `TEXT` |
| `status` | `TEXT` |
| `enrolled_at` | `TIMESTAMPTZ` |

- Sort by `course_title ASC, course_id ASC`.

The generated result type must be:

```go
type ListMemberCoursesRow struct {
    CourseID    uuid.UUID
    CourseTitle string
    Status      string
    EnrolledAt  time.Time
}
```

### JOIN Exercise 2: List a Course Roster

Implement `ListCourseRoster` using `courses`, `enrollments`, and `members`.

The query must:

- Accept one course ID.
- Include only enrollments whose status is `enrolled`.
- Exclude `completed` and `cancelled` enrollments.
- Return exactly these columns and aliases:

| Alias | PostgreSQL Type |
| --- | --- |
| `member_id` | `UUID` |
| `member_name` | `TEXT` |
| `member_email` | `TEXT` |
| `status` | `TEXT` |
| `enrolled_at` | `TIMESTAMPTZ` |

- Sort by `member_name ASC, member_id ASC`.

The generated result type must be:

```go
type ListCourseRosterRow struct {
    MemberID    uuid.UUID
    MemberName  string
    MemberEmail string
    Status      string
    EnrolledAt  time.Time
}
```

### JOIN Exercise 3: Get Enrollment Details

Implement `GetEnrollmentDetail` using all three tables.

The query must:

- Accept `member_id` and `course_id` through `sqlc.arg(...)`.
- Use the `:one` annotation.
- Return `pgx.ErrNoRows` through the generated method when the enrollment does not exist.
- Return exactly these columns and aliases:

| Alias | PostgreSQL Type |
| --- | --- |
| `member_id` | `UUID` |
| `member_name` | `TEXT` |
| `member_email` | `TEXT` |
| `course_id` | `UUID` |
| `course_title` | `TEXT` |
| `course_capacity` | `INTEGER` |
| `status` | `TEXT` |
| `enrolled_at` | `TIMESTAMPTZ` |

The generated result type must be:

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

### JOIN Exercise 4: List Classmates

Implement `ListClassmates` with a self-join on `enrollments` and a join to `members`.

The query must:

- Accept one member ID.
- Find other members who share at least one course with the requested member.
- Require both the requested member's enrollment and the classmate's enrollment to have the status `enrolled`.
- Exclude the requested member from the result.
- Use `DISTINCT` so a classmate who shares multiple courses appears only once.
- Return exactly these columns and aliases:

| Alias | PostgreSQL Type |
| --- | --- |
| `member_id` | `UUID` |
| `member_name` | `TEXT` |
| `member_email` | `TEXT` |

- Sort by `member_name ASC, member_id ASC`.

The generated result type must be:

```go
type ListClassmatesRow struct {
    MemberID    uuid.UUID
    MemberName  string
    MemberEmail string
}
```

### Deliverables

1. Complete the three domain `schema.sql` files and all paired migrations.
2. Add the annotated `queries.sql` file to each domain.
3. Run `./scripts/create_sqlc_full_schema.sh` to create the complete schema and `sqlc.yaml`.
4. Run `sqlc generate`.
5. Submit the Go files produced by `sqlc generate`. Do not create or manually edit generated Go code.
6. Write tests or a small Go program that calls the generated query methods.

### Verification

Demonstrate that:

1. `sqlc generate` completes without errors.
2. `go test ./...` compiles and tests the generated query methods.
3. CRUD operations return the expected rows and affect only their intended targets.
4. A member without enrollments receives no rows from `ListMemberCourses`.
5. A course without active enrollments receives no rows from `ListCourseRoster`.
6. Completed and cancelled enrollments do not appear in `ListCourseRoster`.
7. A classmate who shares multiple courses appears only once in `ListClassmates`.
8. `GetEnrollmentDetail` returns `pgx.ErrNoRows` for a missing enrollment.
9. Duplicate or invalid enrollments are rejected by the constraints implemented in Part I.

## Evaluation

This lab uses pass/fail evaluation. 

### Before Running the Evaluation

1. Start PostgreSQL with the connection used throughout this lab:

   ```text
   postgresql://postgres:password@localhost:5432/postgres?sslmode=disable
   ```

2. Apply all `up` migrations so the `members`, `courses`, and `enrollments` tables exist.
3. Generate the sqlc configuration and Go code:

   ```bash
   ./scripts/create_sqlc_full_schema.sh
   sqlc generate
   ```

4. Run the evaluation:

   ```bash
   go test ./internal/...
   ```

### Passing Criteria

A submission passes when all of the following are true:

- The sqlc-generated packages compile against the required query names, parameters, and result fields.
- The database preflight finds the `members`, `courses`, and `enrollments` tables.
- All 10 member tests, 10 course tests, and 14 enrollment tests pass.
- The provided `queries_test.go` files and generated files have not been manually modified.

### Member Evaluation (10 Tests)

| Area | Evaluated behavior |
| --- | --- |
| CRUD (5) | Create, get, list, update, and delete return the expected member data and affect only the requested row. |
| Errors and constraints (2) | A missing member returns `pgx.ErrNoRows`, and a duplicate email is rejected by a unique constraint. |
| Member courses (2) | Complete enrollment history includes every status, follows the required ordering, and returns an empty list when appropriate. |
| Classmates (1) | Results filter by active enrollment, exclude the requested member, remove duplicates, and follow the required ordering. |

### Course Evaluation (10 Tests)

| Area | Evaluated behavior |
| --- | --- |
| CRUD (5) | Create, get, list, update, and delete return the expected course data and affect only the requested row. |
| Errors and constraints (3) | A missing course returns `pgx.ErrNoRows`; zero and negative capacities are rejected by a check constraint. |
| Course roster (2) | The roster contains only active enrollments, excludes completed and cancelled enrollments, follows the required ordering, and can be empty. |

### Enrollment Evaluation (14 Tests)

| Area | Evaluated behavior |
| --- | --- |
| CRUD (5) | Create, get, list, update status, and delete use the member/course composite identifier and return the expected data. |
| Errors and constraints (5) | Duplicate pairs, missing member/course references, and invalid statuses are rejected; a missing enrollment returns `pgx.ErrNoRows`. |
| Enrollment detail (2) | The detail query returns the required joined member, course, and enrollment fields, or `pgx.ErrNoRows` when missing. |
| Cascades (2) | Deleting either the related member or course automatically removes the enrollment. |

If the generated packages do not compile, first check the query annotations, names, parameters, selected columns, and aliases against the contracts above. A preflight failure means PostgreSQL is unavailable or the migrations have not created all three tables. A failed assertion or unexpected SQLSTATE means the query behavior or database constraint does not match the specification.

-- name: GetAllUsers :many
SELECT * FROM users;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserWithAlias :many
SELECT u.* FROM users u;

-- name: CreateUser :one
INSERT INTO users (name, email, password, ssn)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateUser :one
UPDATE users SET name = $2 WHERE id = $1 RETURNING *;

-- name: DeleteUser :one
DELETE FROM users WHERE id = $1 RETURNING *;

-- name: GetAllAccounts :many
SELECT * FROM accounts;

-- name: GetUserAndAccount :many
SELECT users.*, accounts.* FROM users JOIN accounts ON users.id = accounts.user_id;

-- :dfryer:migrations:Create migrations table(s)
CREATE TABLE migrations (
    id SERIAL PRIMARY KEY,
    namespace VARCHAR(50) NOT NULL,
    user VARCHAR(50),
    comment TEXT,
    ddl TEXT,
    completed TIMESTAMP,
)